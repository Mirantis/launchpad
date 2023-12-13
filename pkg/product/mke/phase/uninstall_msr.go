package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr/msr2"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// UninstallMSR is the phase implementation for running MSR uninstall.
type UninstallMSR struct {
	phase.Analytics
	phase.BasicPhase
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

	if err := msr2.Cleanup(msrHosts, swarmLeader, p.Config); err != nil {
		return fmt.Errorf("failed to cleanup MSR: %w", err)
	}

	return nil
}
