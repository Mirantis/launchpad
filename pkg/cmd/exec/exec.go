package exec

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/exec"

	log "github.com/sirupsen/logrus"
)

// Exec ...
func Exec(configFile string, address string, interactive, first bool, cmd string) error {
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

	var host *api.Host
	if address == "" {
		if first {
			host = clusterConfig.Spec.Hosts.First()
			if host == nil {
				// this should never happen
				return fmt.Errorf("failed to get the first host from configuration")
			}
		} else {
			return fmt.Errorf("--address or --first required") // feels like this is in the wrong place
		}
	} else if address == "localhost" {
		host = clusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			return h.Localhost
		})
	} else if strings.Contains(address, ":") {
		parts := strings.SplitN(address, ":", 2)
		addr := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid port: %s", parts[1])
		}

		host = clusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			if h.Address != addr || h.Localhost {
				return false
			}
			if h.WinRM != nil {
				return h.WinRM.Port == port
			}
			return h.SSH.Port == port
		})
	} else {
		host = clusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			return h.Address == address
		})
	}
	if host == nil {
		return fmt.Errorf("Host with address %s not found in configuration", address)
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

	if err := host.Connect(); err != nil {
		return fmt.Errorf("Failed to connect: %s", err.Error())
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
