package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"

	log "github.com/sirupsen/logrus"
)

// UninstallMSR is the phase implementation for running MSR uninstall
type UninstallMSR struct {
	phase.Analytics
	MSRPhase
}

// Title prints the phase title
func (p *UninstallMSR) Title() string {
	return "Uninstall MSR components"
}

// Run an uninstall via msr.Cleanup
func (p *UninstallMSR) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()
	if !p.Config.Spec.MSR.Metadata.Installed {
		log.Infof("%s: MSR is not installed", swarmLeader)
		return nil
	}

	var msrHosts []*api.Host

	for _, h := range p.Config.Spec.Hosts {
		if h.Role == "msr" {
			msrHosts = append(msrHosts, h)
		}
	}
	return msr.Cleanup(msrHosts, swarmLeader)
}
