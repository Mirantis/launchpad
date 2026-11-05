package mke

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	mcclog "github.com/Mirantis/launchpad/pkg/log"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/util/cmdbuffer"
	"github.com/Mirantis/launchpad/pkg/util/secretenv"
	rigcmd "github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

// BootstrapOptions configure options for the Bootstrap.
type BootstrapOptions struct {
	OperationFlags  commonconfig.Flags  // OPTIONAL: flags to pass to the bootstrapper command
	CleanupDisabled bool                // OPTIONAL: if true, then the bootstrapper container will not be removed
	ExecOptions     []rigcmd.ExecOption // OPTIONAL: additional rig exec options to pass down to rig
	SecretEnv       map[string]string   // OPTIONAL: values staged on the host and passed by environment, keyed by variable name
}

// secretEnvByFlag maps the bootstrapper flags that carry a credential to the
// environment variable it also reads them from. Taken from the MKE install
// CLI reference, which documents the same pairs for 3.3 through 3.9.
//
// Flags reach this table from a user's spec.mke.installFlags as well as from
// launchpad's own phases, so extraction happens here rather than at each
// call site.
var secretEnvByFlag = map[string]string{ // #nosec G101 -- flag names mapped to environment variable names; no value here is a credential
	"--admin-username":    "UCP_ADMIN_USER",
	"--admin-password":    "UCP_ADMIN_PASSWORD",
	"--license":           "UCP_LICENSE",
	"--registry-username": "REGISTRY_USERNAME",
	"--registry-password": "REGISTRY_PASSWORD",
	"--unlock-key":        "UNLOCK_KEY",
}

// Bootstrap a leader host using the MKE bootsrapper as docker run, returning output.
func Bootstrap(operation string, config mkeconfig.ClusterConfig, bootoptions BootstrapOptions) (output string, err error) {
	image := config.Spec.MKE.GetBootstrapperImage()
	leader := config.Spec.SwarmLeader()

	if mcclog.Debug {
		bootoptions.OperationFlags.AddUnlessExist("--debug")
	}

	runFlags := commonconfig.Flags{"-i", "-v /var/run/docker.sock:/var/run/docker.sock"}

	if !bootoptions.CleanupDisabled {
		runFlags.Add("--rm")
	}

	if leader.Configurer.SELinuxEnabled(leader) {
		runFlags.Add("--security-opt label=disable")
	}

	// Secrets are staged in mode-0600 files on the host and read back into
	// the environment when the command runs, so they never enter argv.
	fromFlags, operationFlags := secretenv.Extract(bootoptions.OperationFlags, secretEnvByFlag)
	for name, value := range bootoptions.SecretEnv {
		if value != "" {
			fromFlags[name] = value
		}
	}

	secrets, err := secretenv.Stage(leader, fromFlags)
	if err != nil {
		return output, fmt.Errorf("mke bootstrap secret staging error: %w", err)
	}

	defer func() {
		if rmErr := secrets.Remove(); rmErr != nil {
			log.Warnf("%s: %s", leader, rmErr)
		}
	}()

	for _, flag := range secrets.PassFlags() {
		runFlags.Add(flag)
	}

	opFlags := commonconfig.Flags(operationFlags)
	dockerCmd := leader.Configurer.DockerCmd().Raw("run").Raw(runFlags.Join()).Arg(image).Arg(operation).Raw(opFlags.Join()).String()
	cmd := strings.TrimSpace(secrets.Assignments() + " " + dockerCmd)
	buf := cmdbuffer.NewBuffer() // an io.Reader which .Read() doesn't eof until .eof() is run. On eof is blocks the .Read

	if wait, err := leader.ExecStreams(cmd, nil, buf, buf, bootoptions.ExecOptions...); err != nil {
		return output, fmt.Errorf("mke bootstrap exec error: %w", err)
	} else { //nolint: revive
		go func() {
			if waitErr := wait.Wait(); waitErr != nil {
				err = errors.Join(err, fmt.Errorf("mke bootstrap %s failure; %w", operation, waitErr))
			}
			buf.EOF()
		}()
	}

	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		line := scanner.Text()

		if le, parseErr := cmdbuffer.LogrusParseText(line); parseErr == nil {
			// output was logrus, so pipe it to launchpad logrus
			output += fmt.Sprintf("%s\n", le.Msg)
			le.Msg = fmt.Sprintf("MKE %s: %s", operation, le.Msg)
			cmdbuffer.LogrusLine(le)

			if le.Level == "fatal" {
				err = errors.Join(err, fmt.Errorf("mke bootstrap %s failure; %s", operation, le.Msg))
			}
		} else {
			// output line was not logrus, so just output it
			output += fmt.Sprintf("%s\n", line)
			fmt.Println(line)
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		err = errors.Join(err, fmt.Errorf("mke bootstrap output scan error: %w", scanErr))
	}

	return output, err
}
