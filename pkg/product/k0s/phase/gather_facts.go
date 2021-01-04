package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"

	"github.com/Mirantis/mcc/pkg/product/k0s/api"

	log "github.com/sirupsen/logrus"
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts
type GatherFacts struct {
	phase.Analytics
	BasicPhase
}

// Title for the phase
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel
func (p *GatherFacts) Run() error {
	err := RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.investigateHost)
	if err != nil {
		return err
	}

	return nil
}

func (p *GatherFacts) investigateHost(h *api.Host, c *api.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h)

	h.Metadata = &api.HostMetadata{}

	if err := h.Configurer.CheckPrivilege(); err != nil {
		return err
	}

	version, err := h.K0sVersion()
	if err != nil {
		log.Infof("%s: K0s is not installed", h)
	} else {
		log.Infof("%s: is running k0s version %s", h, version)
		h.Metadata.K0sVersion = version
	}

	log.Infof("%s: is running \"%s\"", h, h.Metadata.Os.Name)

	log.Infof("%s: gathered all facts", h)

	return nil
}
