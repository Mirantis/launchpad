package phase

import (
	"fmt"
	"regexp"
	"strconv"
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
	if err := p.validateDataPlane(conf); err != nil {
		return err
	}

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

// validateDataPlane checks if the calico data plane would get changed (VXLAN <-> VPIP)
func (p *ValidateHosts) validateDataPlane(conf *api.ClusterConfig) error {
	log.Debug("validating data plane settings")

	re := regexp.MustCompile(`^--calico-vxlan=?(.*)`)

	hasTrue := false
	hasFalse := false

	for _, v := range conf.Spec.Ucp.InstallFlags {
		match := re.FindStringSubmatch(v)
		if len(match) == 2 {
			if match[1] == "" {
				hasTrue = true
			} else {
				b, err := strconv.ParseBool(match[1])
				if err != nil {
					return fmt.Errorf("invalid --calico-vxlan value %s", v)
				}
				if b {
					hasTrue = true
				} else {
					hasFalse = true
				}
			}
		}
	}

	// User has explicitly defined --calico-vxlan=false but there is a windows host in the config
	if hasFalse {
		if conf.Spec.Hosts.Include(func(h *api.Host) bool { return h.IsWindows() }) {
			return fmt.Errorf("calico IPIP can't be used on Windows")
		}

		log.Debug("no windows hosts found")
	}

	if !conf.Spec.Ucp.Metadata.Installed {
		log.Debug("no existing UCP installation")
		return nil
	}

	// User has explicitly defined --calico-vxlan=false but there is already a calico with vxlan
	if conf.Spec.Ucp.Metadata.VXLAN {
		log.Debug("ucp has been installed with calico + vxlan")
		if hasFalse {
			return fmt.Errorf("calico configured with VXLAN, can't automatically change to IPIP")
		}
	} else {
		log.Debug("ucp has been installed with calico + vpip")
		// User has explicitly defined --calico-vxlan=true but there is already a calico with ipip
		if hasTrue {
			return fmt.Errorf("calico configured with IPIP, can't automatically change to VXLAN")
		}
	}

	log.Debug("data plane settings check passed")

	return nil
}
