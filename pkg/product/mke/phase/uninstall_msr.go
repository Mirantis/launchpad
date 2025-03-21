package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/msr"
	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// UninstallMSR is the phase implementation for running MSR uninstall.
type UninstallMSR struct {
	phase.Analytics
	MSRPhase
}

// Title prints the phase title.
func (p *UninstallMSR) Title() string {
	return "Uninstall MSR components"
}

// Run an uninstall via msr.Cleanup.
func (p *UninstallMSR) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()
	msrLeader := p.Config.Spec.MSRLeader()
	if msrLeader == nil || !msrLeader.MSRMetadata.Installed {
		log.Infof("%s: MSR is not installed", swarmLeader)
		return nil
	}

	var msrHosts []*api.Host

	for _, h := range p.Config.Spec.Hosts {
		if h.Role == "msr" {
			msrHosts = append(msrHosts, h)
		}
	}
	if err := msr.Cleanup(msrHosts, swarmLeader, p.Config); err != nil {
		return fmt.Errorf("failed to clean up MSR: %w", err)
	}
	return nil
}
