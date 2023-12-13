package phase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/Mirantis/mcc/pkg/msr/msr2"

	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/oracle"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/sles"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init.
	"github.com/k0sproject/dig"

	_ "github.com/Mirantis/mcc/pkg/configurer/windows"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts.
type GatherFacts struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel.
func (p *GatherFacts) Run() error {
	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.investigateHost)
	if err != nil {
		return fmt.Errorf("failed to gather facts: %w", err)
	}
	// Gather MKE related facts

	swarmLeader := p.Config.Spec.SwarmLeader()

	// If engine is installed, we can collect some MKE & Swarm related info too
	if swarmLeader.Metadata.MCRVersion != "" {
		err := mke.CollectFacts(swarmLeader, p.Config.Spec.MKE.Metadata)
		if err != nil {
			return fmt.Errorf("%s: failed to collect existing MKE details: %w", swarmLeader, err)
		}
		if p.Config.Spec.MKE.Metadata.Installed {
			log.Infof("%s: MKE has version %s", swarmLeader, p.Config.Spec.MKE.Metadata.InstalledVersion)
		} else {
			log.Infof("%s: MKE is not installed", swarmLeader)
		}
		p.Config.Spec.MKE.Metadata.ClusterID = swarm.ClusterID(swarmLeader)
	}
	if p.Config.Spec.ContainsMSR() {
		// If we intend to configure msr as well, gather facts for msr
		if p.Config.Spec.MSR == nil {
			p.Config.Spec.MSR = &api.MSRConfig{}
		}

		msrHosts := p.Config.Spec.MSRs()
		_ = msrHosts.ParallelEach(func(h *api.Host) error {
			if h.Metadata != nil && h.Metadata.MCRVersion != "" {
				msrMeta, err := msr2.CollectFacts(h)
				if err != nil {
					log.Debugf("%s: failed to collect existing msr details: %s", h, err.Error())
				}
				h.MSRMetadata = msrMeta
				if msrMeta.Installed {
					log.Infof("%s: msr has version %s", h, msrMeta.InstalledVersion)
				} else {
					log.Infof("%s: msr is not installed", h)
				}
			}
			return nil
		})
	}

	return nil
}

var errInvalidIP = errors.New("invalid IP address")

func (p *GatherFacts) investigateHost(h *api.Host, _ *api.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h)
	if h.Metadata == nil {
		h.Metadata = &api.HostMetadata{}
	}

	if err := h.Configurer.CheckPrivilege(h); err != nil {
		return fmt.Errorf("privilege check failed: %w", err)
	}

	version, err := h.MCRVersion()
	if err != nil || version == "" {
		log.Infof("%s: mirantis container runtime not installed", h)
	} else {
		log.Infof("%s: is running mirantis container runtime version %s", h, version)
		configData, err := h.Configurer.ReadFile(h, "/etc/docker/daemon.json")
		if err == nil {
			var newCfg dig.Mapping
			if err = json.Unmarshal([]byte(configData), &newCfg); err == nil {
				for k, v := range newCfg {
					if _, ok := h.DaemonConfig[k]; !ok {
						log.Debugf("%s: set %s = %t for spec.hosts[].daemonConfig from existing daemon.json", h, k, v)
						h.DaemonConfig[k] = v
					}
				}
			}
		}
	}

	h.Metadata.MCRVersion = version

	h.Metadata.Hostname = h.Configurer.Hostname(h)
	h.Metadata.LongHostname = h.Configurer.LongHostname(h)

	if h.PrivateInterface == "" {
		i, err := h.Configurer.ResolvePrivateInterface(h)
		if err != nil {
			return fmt.Errorf("%s: failed to resolve private interface: %w", h, err)
		}
		log.Infof("%s: detected private interface '%s'", h, i)
		h.PrivateInterface = i
	}

	a, err := h.Configurer.ResolveInternalIP(h, h.PrivateInterface, h.Address())
	if err != nil {
		return fmt.Errorf("%s: failed to resolve internal address: %w", h, err)
	}
	if net.ParseIP(a) == nil {
		return fmt.Errorf("%s: %w: failed to resolve internal address: invalid IP address: %q", h, errInvalidIP, a)
	}
	h.Metadata.InternalAddress = a

	log.Infof("%s: is running \"%s\"", h, h.OSVersion.String())
	log.Infof("%s: internal address: %s", h, h.Metadata.InternalAddress)

	log.Infof("%s: gathered all facts", h)

	return nil
}
