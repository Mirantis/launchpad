package mke

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

var errInvalidTarget = errors.New("invalid target")

// Exec runs commands or shell sessions on a configuration host.
func (p *MKE) Exec(targets []string, interactive, first, all, parallel bool, role, hostos, cmd string) error { //nolint:maintidx
	var hosts config.Hosts

	for _, target := range targets {
		switch {
		case target == "localhost":
			hosts = append(hosts, &config.Host{Connection: rig.Connection{Localhost: &rig.Localhost{Enabled: true}}})
		case strings.Contains(target, ":"):
			parts := strings.SplitN(target, ":", 2)
			addr := parts[0]
			port, err := strconv.Atoi(parts[1])
			if err != nil {
				return fmt.Errorf("%w: invalid port: %s", errInvalidTarget, parts[1])
			}

			host := p.ClusterConfig.Spec.Hosts.Find(func(h *config.Host) bool {
				if h.Address() != addr {
					return false
				}
				if h.WinRM != nil {
					return h.WinRM.Port == port
				}
				return h.SSH.Port == port
			})
			if host == nil {
				return fmt.Errorf("%w: host %s not found in configuration", errInvalidTarget, target)
			}
			hosts = append(hosts, host)
		default:
			host := p.ClusterConfig.Spec.Hosts.Find(func(h *config.Host) bool {
				return h.Address() == target
			})
			if host == nil {
				return fmt.Errorf("%w: host %s not found in configuration", errInvalidTarget, target)
			}
			hosts = append(hosts, host)
		}
	}

	if role != "" {
		if len(hosts) == 0 {
			hosts = p.ClusterConfig.Spec.Hosts.Filter(func(h *config.Host) bool { return h.Role == role })
		} else {
			hosts = hosts.Filter(func(h *config.Host) bool { return h.Role == role })
		}
	}

	if hostos != "" {
		if len(hosts) == 0 {
			hosts = p.ClusterConfig.Spec.Hosts
		}

		var foundhosts config.Hosts
		var mutex sync.Mutex

		err := hosts.ParallelEach(func(h *config.Host) error {
			if err := h.Connect(); err != nil {
				return fmt.Errorf("failed to connect to host %s: %w", h.Address(), err)
			}
			if err := h.ResolveConfigurer(); err != nil {
				return fmt.Errorf("failed to resolve configurer for host %s: %w", h.Address(), err)
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
			return fmt.Errorf("failed to filter hosts by OS: %w", err)
		}
		hosts = foundhosts
	}

	if role == "" && hostos == "" && len(targets) == 0 {
		hosts = p.ClusterConfig.Spec.Hosts
	}

	if first {
		if len(hosts) == 0 {
			return fmt.Errorf("%w: no hosts found but --first given", errInvalidTarget)
		}
		hosts = hosts[0:1]
	}

	if len(hosts) > 1 {
		if !all {
			return fmt.Errorf("%w: found %d hosts but --all not given", errInvalidTarget, len(hosts))
		}
		if interactive {
			return fmt.Errorf("%w: can't use --interactive with multiple targets", errInvalidTarget)
		}
	}

	if len(hosts) == 0 {
		println("no hosts found")
		return nil
	}

	var stdin string

	if !interactive {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat stdin: %w", err)
		}

		if (stat.Mode() & os.ModeCharDevice) == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			stdin = string(data)
		}
	}

	err := hosts.ParallelEach(func(h *config.Host) error {
		if err := h.Connect(); err != nil {
			return fmt.Errorf("connect to host %s: %w", h.Address(), err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to connect to hosts: %w", err)
	}

	var linuxcount, windowscount int
	err = hosts.Each(func(h *config.Host) error {
		if h.IsWindows() {
			if linuxcount > 0 {
				return fmt.Errorf("%w mixed target operating systems, use --os linux or --os windows", errInvalidTarget)
			}
			windowscount++
		} else {
			if windowscount > 0 {
				return fmt.Errorf("%w: mixed target operating systems, use --os linux or --os windows", errInvalidTarget)
			}
			linuxcount++
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("target operating system check failed: %w", err)
	}

	if cmd == "" {
		if stdin != "" {
			return fmt.Errorf("%w: can't pipe to a remote shell without a command", errInvalidTarget)
		}
		log.Tracef("assuming intention to run a shell with --interactive")
		err := hosts[0].ExecInteractive("")
		if err != nil {
			return fmt.Errorf("failed to run interactive shell: %w", err)
		}
	}

	if interactive {
		log.Tracef("running interactive with cmd: %q", cmd)
		if err := hosts[0].ExecInteractive(cmd); err != nil {
			return fmt.Errorf("failed to run interactive shell: %w", err)
		}
		return nil
	}

	log.Tracef("running non-interactive with cmd: %q", cmd)
	runFunc := func(h *config.Host) error {
		if err := h.Exec(cmd, exec.Stdin(stdin), exec.StreamOutput()); err != nil {
			return fmt.Errorf("failed on host %s: %w", h.Address(), err)
		}
		return nil
	}
	if parallel {
		err = hosts.ParallelEach(runFunc)
	} else {
		err = hosts.Each(runFunc)
	}

	if err != nil {
		return fmt.Errorf("failed to run command on hosts: %w", err)
	}

	return nil
}
