package mke

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// Exec runs commands or shell sessions on a configuration host.
func (p *MKE) Exec(targets []string, interactive, first, all, parallel bool, role, hostos, cmd string) error {
	var hosts api.Hosts

	for _, target := range targets {
		if target == "localhost" {
			hosts = append(hosts, &api.Host{Connection: rig.Connection{Localhost: &rig.Localhost{Enabled: true}}})
		} else if strings.Contains(target, ":") {
			parts := strings.SplitN(target, ":", 2)
			addr := parts[0]
			port, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("invalid port: %s", parts[1])
			}

			host := p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
				if h.Address() != addr {
					return false
				}
				if h.WinRM != nil {
					return h.WinRM.Port == port
				}
				return h.SSH.Port == port
			})
			if host == nil {
				return fmt.Errorf("host %s not found in configuration", target)
			}
			hosts = append(hosts, host)
		} else {
			host := p.ClusterConfig.Spec.Hosts.Find(func(h *api.Host) bool {
				return h.Address() == target
			})
			if host == nil {
				return fmt.Errorf("host %s not found in configuration", target)
			}
			hosts = append(hosts, host)
		}
	}

	if role != "" {
		if len(hosts) == 0 {
			hosts = p.ClusterConfig.Spec.Hosts.Filter(func(h *api.Host) bool { return h.Role == role })
		} else {
			hosts = hosts.Filter(func(h *api.Host) bool { return h.Role == role })
		}
	}

	if hostos != "" {
		if len(hosts) == 0 {
			hosts = p.ClusterConfig.Spec.Hosts
		}

		var foundhosts api.Hosts
		var mutex sync.Mutex

		err := hosts.ParallelEach(func(h *api.Host) error {
			if err := h.Connect(); err != nil {
				return err
			}
			if err := h.ResolveConfigurer(); err != nil {
				return err
			}
			if h.IsWindows() {
				if hostos == "windows" {
					mutex.Lock()
					foundhosts = append(foundhosts, h)
					mutex.Unlock()
				}
			} else {
				if hostos == "linux" || h.OSVersion.ID == hostos {
					mutex.Lock()
					foundhosts = append(foundhosts, h)
					mutex.Unlock()
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		hosts = foundhosts
	}

	if role == "" && hostos == "" && len(targets) == 0 {
		hosts = p.ClusterConfig.Spec.Hosts
	}

	if first {
		if len(hosts) == 0 {
			return fmt.Errorf("no hosts found but --first given")
		}
		hosts = hosts[0:1]
	}

	if len(hosts) > 1 {
		if !all {
			return fmt.Errorf("found %d hosts but --all not given", len(hosts))
		}
		if interactive {
			return fmt.Errorf("can't use --interactive with multiple targets")
		}
	}

	if len(hosts) == 0 {
		println("no hosts found")
		return nil
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
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		stdin = string(data)
	}

	if err := hosts.ParallelEach(func(h *api.Host) error { return h.Connect() }); err != nil {
		return err
	}

	var linuxcount, windowscount int
	hosts.Each(func(h *api.Host) error {
		if h.IsWindows() {
			if linuxcount > 0 {
				return fmt.Errorf("mixed target operating systems, use --os linux or --os windows")
			}
			windowscount++
		} else {
			if windowscount > 0 {
				return fmt.Errorf("mixed target operating systems, use --os linux or --os windows")
			}
			linuxcount++
		}
		return nil
	})

	if cmd == "" {
		if stdin != "" {
			return fmt.Errorf("can't pipe to a remote shell without a command")
		}
		log.Tracef("assuming intention to run a shell with --interactive")
		return hosts[0].Connection.ExecInteractive("")
	}

	if interactive {
		log.Tracef("running interactive with cmd: %q", cmd)
		return hosts[0].Connection.ExecInteractive(cmd)
	}

	log.Tracef("running non-interactive with cmd: %q", cmd)
	if parallel {
		return hosts.ParallelEach(func(h *api.Host) error { return h.Exec(cmd, exec.Stdin(stdin), exec.StreamOutput()) })
	}
	return hosts.Each(func(h *api.Host) error { return h.Exec(cmd, exec.Stdin(stdin), exec.StreamOutput()) })
}
