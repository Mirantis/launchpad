package phase

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	mcclog "github.com/Mirantis/launchpad/pkg/log"
	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/Mirantis/launchpad/pkg/util/stringutil"
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

var errValidationFailed = fmt.Errorf("validation failed")

func (p *ValidateHosts) formatErrors() error {
	errorHosts := p.Config.Spec.Hosts.Filter(func(h *api.Host) bool { return h.Errors.Count() > 0 })

	if len(errorHosts) > 0 {
		messages := errorHosts.MapString(func(h *api.Host) string {
			return fmt.Sprintf("%s:\n%s\n", h, h.Errors.String())
		})

		return fmt.Errorf("%w: %d of %d hosts failed validation:\n%s", errValidationFailed, len(errorHosts), len(p.Config.Spec.Hosts), strings.Join(messages, "\n"))
	}

	return nil
}

var errContentMismatch = fmt.Errorf("content mismatch")

func (p *ValidateHosts) validateHostConnection() error {
	testFile, err := os.CreateTemp("", "uploadTest")
	if err != nil {
		return fmt.Errorf("connection test failed: create temp file: %w", err)
	}
	defer os.Remove("uploadTest")

	_, err = io.CopyN(testFile, rand.Reader, 1048576) // create an 1MB temp file full of random data
	if err != nil {
		return fmt.Errorf("connection test failed: write test file content: %w", err)
	}
	// TODO: validate content

	err = p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		log.Infof("%s: testing file upload", h)
		defer func() {
			if err := h.Configurer.DeleteFile(h, "launchpad.test"); err != nil {
				log.Debugf("%s: failed to delete test file: %s", h, err.Error())
			}
		}()
		err := h.WriteFileLarge(testFile.Name(), h.Configurer.JoinPath(h.Configurer.Pwd(h), "launchpad.test"), fs.FileMode(0o640))
		if err != nil {
			h.Errors.Add(err.Error())
		}
		return fmt.Errorf("failed to upload file: %w", err)
	})
	if err != nil {
		return fmt.Errorf("connection test failed: upload: %w", err)
	}

	err = p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		filename := "launchpad.test"
		testStr := "hello world!\n"
		defer func() {
			if err := h.Configurer.DeleteFile(h, filename); err != nil {
				log.Debugf("%s: failed to delete test file: %s", h, err.Error())
			}
		}()
		log.Infof("%s: testing stdin redirection", h)
		if h.IsWindows() {
			err := h.Exec(fmt.Sprintf(`findstr "^" > %s`, filename), exec.Stdin(testStr))
			if err != nil {
				return fmt.Errorf("failed to test stdin redirection: %w", err)
			}
		} else {
			err := h.Exec(fmt.Sprintf("cat > %s", filename), exec.Stdin(testStr))
			if err != nil {
				return fmt.Errorf("failed to test stdin redirection: %w", err)
			}
		}
		content, err := h.Configurer.ReadFile(h, filename)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if strings.TrimSpace(content) != strings.TrimSpace(testStr) {
			// Allow trailing linefeeds etc, mainly because windows is weird.
			return fmt.Errorf("%w: file write test content check mismatch: %q vs %q", errContentMismatch, strings.TrimSpace(content), strings.TrimSpace(testStr))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	return nil
}

func (p *ValidateHosts) validateLocalhost() {
	_ = p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		if err := h.Configurer.ValidateLocalhost(h); err != nil {
			h.Errors.Add(err.Error())
		}
		return nil
	})
}

func (p *ValidateHosts) validateHostLocalAddresses() {
	_ = p.Config.Spec.Hosts.ParallelEach(p.validateHostLocalAddress)
}

func (p *ValidateHosts) validateHostLocalAddress(h *api.Host) error {
	localAddresses, err := h.Configurer.LocalAddresses(h)
	if err != nil {
		h.Errors.Add(fmt.Sprintf("failed to find host local addresses: %s", err.Error()))
		return nil
	}

	if !stringutil.StringSliceContains(localAddresses, h.Metadata.InternalAddress) {
		msg := fmt.Sprintf("discovered private address %s does not seem to be a node local address (%s). Make sure you've set correct 'privateInterface' for the host in config", h.Metadata.InternalAddress, strings.Join(localAddresses, ","))
		h.Errors.Add(msg)
		return nil
	}

	return nil
}

func (p *ValidateHosts) validateHostnameUniqueness() {
	log.Infof("validating hostname uniqueness")
	hostnames := make(map[string]api.Hosts)

	_ = p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		hostnames[h.Metadata.Hostname] = append(hostnames[h.Metadata.Hostname], h)
		return nil
	})

	for hn, hosts := range hostnames {
		if len(hosts) > 1 {
			others := strings.Join(hosts.MapString(func(h *api.Host) string { return h.Address() }), ", ")
			_ = hosts.Each(func(h *api.Host) error {
				h.Errors.Addf("duplicate hostname '%s' found on hosts %s", hn, others)
				return nil
			})
		}
	}
}

func (p *ValidateHosts) validateDockerGroup() {
	_ = p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		if !h.IsLocal() || h.IsWindows() {
			return nil
		}

		if err := h.Exec("getent group docker"); err != nil {
			h.Errors.Addf("group 'docker' required to exist when running on localhost connection")
			return nil
		}

		if h.Exec(`[ "$(id -u)" = 0 ]`) == nil {
			return nil
		}

		if err := h.Exec("groups | grep -q docker"); err != nil {
			log.Errorf("%s: user must be root or a member of the group 'docker' when running on localhost connection.", h)
			log.Errorf("%s: use 'sudo groupadd -f -g 999 docker && sudo usermod -aG docker $USER' and re-login before running launchpad again.", h)
			h.Errors.Addf("user must be root or a member of the group 'docker'")
		}

		return nil
	})
}
