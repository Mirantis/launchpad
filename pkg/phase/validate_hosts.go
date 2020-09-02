package phase

import (
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"

	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init
	_ "github.com/Mirantis/mcc/pkg/configurer/windows"
	log "github.com/sirupsen/logrus"
)

// ValidateHosts phase implementation to collect facts (OS, version etc.) from hosts
type ValidateHosts struct {
	Analytics
}

// Title for the phase
func (p *ValidateHosts) Title() string {
	return "Validate Hosts"
}

// Run collect all the facts from hosts in parallel
func (p *ValidateHosts) Run(conf *api.ClusterConfig) error {
	if err := p.validateHostFacts(conf); err != nil {
		return p.formatErrors(conf)
	}

	if err := p.validateHostnameUniqueness(conf); err != nil {
		return p.formatErrors(conf)
	}

	return p.formatErrors(conf)
}

func (p *ValidateHosts) formatErrors(conf *api.ClusterConfig) error {
	errorHosts := conf.Spec.Hosts.Filter(func(h *api.Host) bool { return h.Errors.Count() > 0 })

	if len(errorHosts) > 0 {
		messages := errorHosts.MapString(func(h *api.Host) string {
			return fmt.Sprintf("%s:\n%s\n", h.Address, h.Errors.String())
		})

		return fmt.Errorf("%d of %d hosts failed validation:\n%s", len(errorHosts), len(conf.Spec.Hosts), strings.Join(messages, ""))
	}

	return nil
}

func (p *ValidateHosts) validateHostFacts(conf *api.ClusterConfig) error {
	return conf.Spec.Hosts.ParallelEach(func(h *api.Host) error {
		log.Infof("%s: validating host facts", h.Address)
		err := h.Configurer.ValidateFacts()
		if err != nil {
			h.Errors.Add(err.Error())
			return err
		}
		return nil
	})
}

func (p *ValidateHosts) validateHostnameUniqueness(conf *api.ClusterConfig) error {
	log.Infof("validating hostname uniqueness")
	hostnames := make(map[string]api.Hosts)

	conf.Spec.Hosts.Each(func(h *api.Host) error {
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

	return p.formatErrors(conf)
}
