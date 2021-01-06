package mke

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/k0sproject/rig/exec"

	log "github.com/sirupsen/logrus"
)

// Exec runs commands or shell sessions on a configuration host
func (p *MKE) Exec(target string, interactive, first bool, role, cmd string) error {
	var host *api.Host
	if target == "" {
		if role != "" {
			host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool { return h.Role == role })
			if host == nil {
				return fmt.Errorf("failed to get the first host with role '%s' from configuration", role)
			}
		} else if first {
			host = p.ClusterConfig.Spec.Hosts.First()
			if host == nil {
				return fmt.Errorf("failed to get the first host from configuration")
			}
		} else {
			return fmt.Errorf("--target, --first or --role required") // feels like this is in the wrong place
		}
	} else if target == "localhost" {
		host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			return h.Localhost
		})
	} else if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		addr := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("invalid port: %s", parts[1])
		}

		host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			if h.Address != addr || h.Localhost {
				return false
			}
			if h.WinRM != nil {
				return h.WinRM.Port == port
			}
			return h.SSH.Port == port
		})
	} else {
		host = p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
			return h.Address == target
		})
	}
	if host == nil {
		return fmt.Errorf("Host with target %s not found in configuration", target)
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

	log.Debugf("%s: connected", host)

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
