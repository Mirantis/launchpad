package phase

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	"crypto/rand"

	log "github.com/sirupsen/logrus"
)

// ValidateHosts phase implementation to collect facts (OS, version etc.) from hosts
type ValidateHosts struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *ValidateHosts) Title() string {
	return "Validate Hosts"
}

// Run collect all the facts from hosts in parallel
func (p *ValidateHosts) Run() error {
	if mcclog.Trace {
		if err := p.validateHostConnection(); err != nil {
			return p.formatErrors(p.Config)
		}
	}

	if err := p.validateHostFacts(); err != nil {
		return p.formatErrors(p.Config)
	}

	if err := p.validateHostnameUniqueness(); err != nil {
		return p.formatErrors(p.Config)
	}

	return p.formatErrors(p.Config)
}

func (p *ValidateHosts) formatErrors(conf *api.ClusterConfig) error {
	errorHosts := conf.Spec.Hosts.Filter(func(h *api.Host) bool { return h.Errors.Count() > 0 })

	if len(errorHosts) > 0 {
		messages := errorHosts.MapString(func(h *api.Host) string {
			return fmt.Sprintf("%s:\n%s\n", h, h.Errors.String())
		})

		return fmt.Errorf("%d of %d hosts failed validation:\n%s", len(errorHosts), len(conf.Spec.Hosts), strings.Join(messages, ""))
	}

	return nil
}

func (p *ValidateHosts) validateHostConnection() error {
	f, err := ioutil.TempFile("", "uploadTest")
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
		defer h.Configurer.DeleteFile("launchpad.test")
		err := h.WriteFileLarge(f.Name(), h.Configurer.JoinPath(h.Configurer.Pwd(), "launchpad.test"))
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
		defer h.Configurer.DeleteFile(fn)
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
		content, err := h.Configurer.ReadFile(fn)
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

func (p *ValidateHosts) validateHostFacts() error {
	return p.Config.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		log.Infof("%s: validating host facts", h)
		if err := h.Configurer.ValidateFacts(); err != nil {
			h.Errors.Add(err.Error())
			return err
		}
		return nil
	})
}

func (p *ValidateHosts) validateHostnameUniqueness() error {
	log.Infof("validating hostname uniqueness")
	hostnames := make(map[string]api.Hosts)

	p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		hostnames[h.Metadata.Hostname] = append(hostnames[h.Metadata.Hostname], h)
		return nil
	})

	for hn, hosts := range hostnames {
		if len(hosts) > 1 {
			others := strings.Join(hosts.MapString(func(h *api.Host) string { return h.Address }), ", ")
			hosts.Each(func(h *api.Host) error {
				h.Errors.Addf("duplicate hostname '%s' found on hosts %s", hn, others)
				return nil
			})
		}
	}

	return p.formatErrors(p.Config)
}
