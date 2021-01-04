package phase

import (
	"fmt"
	"net"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"

	log "github.com/sirupsen/logrus"
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts
type GatherFacts struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel
func (p *GatherFacts) Run() error {
	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.investigateHost)
	if err != nil {
		return err
	}
	// Gather MKE related facts

	swarmLeader := p.Config.Spec.SwarmLeader()

	// If engine is installed, we can collect some MKE & Swarm related info too
	if swarmLeader.Metadata.MCRVersion != "" {
		err := mke.CollectFacts(swarmLeader, p.Config.Spec.MKE.Metadata)
		if err != nil {
			return fmt.Errorf("%s: failed to collect existing MKE details: %s", swarmLeader, err.Error())
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
		msrHosts.ParallelEach(func(h *api.Host) error {
			if h.Metadata != nil && h.Metadata.MCRVersion != "" {
				msrMeta, err := msr.CollectFacts(h)
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

func (p *GatherFacts) investigateHost(h *api.Host, c *api.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h)

	h.Metadata = &api.HostMetadata{}

	if err := h.Configurer.CheckPrivilege(); err != nil {
		return err
	}

	version, err := h.MCRVersion()
	if err != nil || version == "" {
		log.Infof("%s: mirantis container runtime not installed", h)
	} else {
		log.Infof("%s: is running mirantis container runtime version %s", h, version)
	}

	h.Metadata.MCRVersion = version

	h.Metadata.Hostname = h.Configurer.ResolveHostname()
	h.Metadata.LongHostname = h.Configurer.ResolveLongHostname()

	if h.PrivateInterface == "" {
		i, err := h.Configurer.ResolvePrivateInterface()
		if err != nil {
			return err
		}
		log.Infof("%s: detected private interface '%s'", h, i)
		h.PrivateInterface = i
	}

	a, err := h.Configurer.ResolveInternalIP(h.PrivateInterface, h.Address)
	if err != nil {
		return fmt.Errorf("%s: failed to resolve internal address: %s", h, err.Error())
	}
	if net.ParseIP(a) == nil {
		return fmt.Errorf("%s: failed to resolve internal address: invalid IP address: %q", h, a)
	}
	h.Metadata.InternalAddress = a

	log.Infof("%s: is running \"%s\"", h, h.Metadata.Os.Name)
	log.Infof("%s: internal address: %s", h, h.Metadata.InternalAddress)

	log.Infof("%s: gathered all facts", h)

	return nil
}
