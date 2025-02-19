package mke

import (
	"bufio"
	"fmt"

	mcclog "github.com/Mirantis/launchpad/pkg/log"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/Mirantis/launchpad/pkg/util/cmdbuffer"
	"github.com/k0sproject/rig/exec"
	"github.com/sirupsen/logrus"
)

// BootstrapConfig configure options for the Bootstrap.
type BootstrapConfig struct {
	Config          api.ClusterConfig // REQUIRED: the cluster config is needed
	Operation       string            // REQUIRED: the MKE bootstrapper operation to run (e.g. install, reset)
	OperationFlags  common.Flags      // OPTIONAL: flags to pass to the bootstrapper command
	CleanupDisabled bool              // OPTIONAL: if true, then the bootstrapper container will not be removed
	ExecOptions     []exec.Option     // OPTIONAL: additional rig exec options to pass down to rig
}

// Bootstrap a leader host using the MKE bootsrapper as docker run, returning output.
func Bootstrap(bootconf BootstrapConfig) (string, error) {
	image := bootconf.Config.Spec.MKE.GetBootstrapperImage()
	leader := bootconf.Config.Spec.SwarmLeader()

	if mcclog.Debug {
		bootconf.OperationFlags.AddUnlessExist("--debug")
	}

	runFlags := common.Flags{"-i", "-v /var/run/docker.sock:/var/run/docker.sock"}

	if !bootconf.CleanupDisabled {
		runFlags.Add("--rm")
	}

	if leader.Configurer.SELinuxEnabled(leader) {
		runFlags.Add("--security-opt label=disable")
	}

	cmd := leader.Configurer.DockerCommandf("run %s %s %s %s", runFlags.Join(), image, bootconf.Operation, bootconf.OperationFlags.Join())
	output := ""
	buf := cmdbuffer.NewBuffer() // an io.Reader which .Read() doesn't eof until .eof() is run. On eof is blocks the .Read

	if wait, err := leader.ExecStreams(cmd, nil, buf, buf, bootconf.ExecOptions...); err != nil {
		return output, fmt.Errorf("mke bootstrap exec error: %w", err)
	} else { //nolint: revive
		go func() {
			if err := wait.Wait(); err != nil {
				logrus.Error(err)
			}
			buf.EOF()
		}()
	}

	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		line := sc.Text()
		output += line

		cmdbuffer.LogrusLine(cmdbuffer.LogrusParseText(line))
	}
	if err := sc.Err(); err != nil {
		return output, fmt.Errorf("mke bootstrap output scan error: %w", err)
	}

	return output, nil
}
