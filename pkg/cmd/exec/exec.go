package exec

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/exec"

	log "github.com/sirupsen/logrus"
)

// Exec ...
func Exec(configFile string, address string, interactive bool, cmd string) error {
	cfgData, err := config.ResolveClusterFile(configFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err := config.Validate(&clusterConfig); err != nil {
		return err
	}

	var stdin string

	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		if interactive {
			return fmt.Errorf("--interactive given but there's piped data in stdin")
		}
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		stdin = string(data)
	}

	host := clusterConfig.Spec.Hosts.Find(func(h *api.Host) bool { return h.Address == address })
	if host == nil {
		return fmt.Errorf("Host with address %s not found in configuration", address)
	}

	if err := host.Connect(); err != nil {
		println(fmt.Sprintf("Failed to connect: %s", err.Error()))
		return err
	}

	log.Debugf("%s: connected", host.Address)

	if cmd == "" {
		if stdin != "" {
			return fmt.Errorf("can't pipe to a remote shell without a command")
		}
		log.Tracef("assuming intention to run a shell with --interactive")
		return host.Connection.ExecInteractive("")
	}

	if interactive {
		log.Tracef("running interactive with cmd: %q", cmd)
		return host.Connection.ExecInteractive(cmd)
	}

	log.Tracef("running non-interactive with cmd: %q", cmd)
	return host.Exec(cmd, exec.Stdin(stdin), exec.StreamOutput())
}
