package mke

import (
	"bufio"
	"errors"
	"fmt"

	mcclog "github.com/Mirantis/launchpad/pkg/log"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/util/cmdbuffer"
	"github.com/k0sproject/rig/exec"
)

// BootstrapOptions configure options for the Bootstrap.
type BootstrapOptions struct {
	OperationFlags  commonconfig.Flags // OPTIONAL: flags to pass to the bootstrapper command
	CleanupDisabled bool               // OPTIONAL: if true, then the bootstrapper container will not be removed
	ExecOptions     []exec.Option      // OPTIONAL: additional rig exec options to pass down to rig
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

	cmd := leader.Configurer.DockerCommandf("run %s %s %s %s", runFlags.Join(), image, operation, bootoptions.OperationFlags.Join())
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
				err = errors.Join(err, fmt.Errorf("mke bootstrap %s failure; %s", operation, le.Msg)) //nolint
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
