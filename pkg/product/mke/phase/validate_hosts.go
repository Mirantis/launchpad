package phase

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"strings"

	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// ValidateHosts phase implementation to collect facts (OS, version etc.) from hosts.
type ValidateHosts struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *ValidateHosts) Title() string {
	return "Validate Hosts"
}

// Run collect all the facts from hosts in parallel.
func (p *ValidateHosts) Run() error {
	if mcclog.Trace {
		if err := p.validateHostConnection(); err != nil {
			return p.formatErrors()
		}
	}

	p.validateDockerGroup()
	p.validateHostLocalAddresses()
	p.validateHostnameUniqueness()
	p.validateLocalhost()

	return p.formatErrors()
}

func (p *ValidateHosts) formatErrors() error {
	errorHosts := p.Config.Spec.Hosts.Filter(func(h *api.Host) bool { return h.Errors.Count() > 0 })

	if len(errorHosts) > 0 {
		messages := errorHosts.MapString(func(h *api.Host) string {
			return fmt.Sprintf("%s:\n%s\n", h, h.Errors.String())
		})

		return fmt.Errorf("%d of %d hosts failed validation:\n%s", len(errorHosts), len(p.Config.Spec.Hosts), strings.Join(messages, "\n"))
	}

	return nil
}

func (p *ValidateHosts) validateHostConnection() error {
	f, err := os.CreateTemp("", "uploadTest")
	if err != nil {
		return err
	}
	defer os.Remove("uploadTest")

	_, err = io.CopyN(f, rand.Reader, 1048576) // create an 1MB temp file full of random data
	if err != nil {
		return err
	}
	// TODO: validate content

	err = p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		log.Infof("%s: testing file upload", h)
		defer h.Configurer.DeleteFile(h, "launchpad.test")
		err := h.WriteFileLarge(f.Name(), h.Configurer.JoinPath(h.Configurer.Pwd(h), "launchpad.test"))
		if err != nil {
			h.Errors.Add(err.Error())
		}
		return err
	})
	if err != nil {
		return err
	}

	return p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		fn := "launchpad.test"
		testStr := "hello world!\n"
		defer h.Configurer.DeleteFile(h, fn)
		log.Infof("%s: testing stdin redirection", h)
		if h.IsWindows() {
			err := h.Exec(fmt.Sprintf(`findstr "^" > %s`, fn), exec.Stdin(testStr))
			if err != nil {
				return err
			}
		} else {
			err := h.Exec(fmt.Sprintf("cat > %s", fn), exec.Stdin(testStr))
			if err != nil {
				return err
			}
		}
		content, err := h.Configurer.ReadFile(h, fn)
		if err != nil {
			return err
		}
		if strings.TrimSpace(content) != strings.TrimSpace(testStr) {
			// Allow trailing linefeeds etc, mainly because windows is weird.
			return fmt.Errorf("file write test content check mismatch: %q vs %q", strings.TrimSpace(content), strings.TrimSpace(testStr))
		}
		return nil
	})
}

func (p *ValidateHosts) validateLocalhost() {
	p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		if err := h.Configurer.ValidateLocalhost(h); err != nil {
			h.Errors.Add(err.Error())
		}
		return nil
	})
}

func (p *ValidateHosts) validateHostLocalAddresses() {
	p.Config.Spec.Hosts.ParallelEach(p.validateHostLocalAddress)
}

func (p *ValidateHosts) validateHostLocalAddress(h *api.Host) error {
	localAddresses, err := h.Configurer.LocalAddresses(h)
	if err != nil {
		h.Errors.Add(fmt.Sprintf("failed to find host local addresses: %s", err.Error()))
		return err
	}

	if !util.StringSliceContains(localAddresses, h.Metadata.InternalAddress) {
		h.Errors.Add(fmt.Sprintf("discovered private address %s does not seem to be a node local address (%s). Make sure you've set correct 'privateInterface' for the host in config", h.Metadata.InternalAddress, strings.Join(localAddresses, ",")))
		return err
	}

	return nil
}

func (p *ValidateHosts) validateHostnameUniqueness() {
	log.Infof("validating hostname uniqueness")
	hostnames := make(map[string]api.Hosts)

	p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		hostnames[h.Metadata.Hostname] = append(hostnames[h.Metadata.Hostname], h)
		return nil
	})

	for hn, hosts := range hostnames {
		if len(hosts) > 1 {
			others := strings.Join(hosts.MapString(func(h *api.Host) string { return h.Address() }), ", ")
			hosts.Each(func(h *api.Host) error {
				h.Errors.Addf("duplicate hostname '%s' found on hosts %s", hn, others)
				return nil
			})
		}
	}
}

func (p *ValidateHosts) validateDockerGroup() {
	p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		if !h.IsLocal() || h.IsWindows() {
			return nil
		}

		if err := h.Exec("getent group docker"); err != nil {
			return fmt.Errorf("group 'docker' required to exist when running on localhost connection")
		}

		if h.Exec(`[ "$(id -u)" = 0 ]`) == nil {
			return nil
		}

		if err := h.Exec("groups | grep -q docker"); err != nil {
			log.Errorf("%s: user must be root or a member of the group 'docker' when running on localhost connection.", h)
			log.Errorf("%s: use 'sudo groupadd -f -g 999 docker && sudo usermod -aG docker $USER' and re-login before running launchpad again.", h)
			return fmt.Errorf("user must be root or a member of the group 'docker'")
		}

		return nil
	})
}
