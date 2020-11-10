package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/dtr"
	"github.com/Mirantis/mcc/pkg/phase"

	log "github.com/sirupsen/logrus"
)

// UninstallDTR is the phase implementation for running DTR uninstall
type UninstallDTR struct {
	phase.Analytics
	DtrPhase
}

// Title prints the phase title
func (p *UninstallDTR) Title() string {
	return "Uninstall DTR components"
}

// Run an uninstall via dtr.Cleanup
func (p *UninstallDTR) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()
	if !p.Config.Spec.Dtr.Metadata.Installed {
		log.Infof("%s: DTR is not installed", swarmLeader)
		return nil
	}

	var dtrHosts []*api.Host

	for _, h := range p.Config.Spec.Hosts {
		if h.Role == "dtr" {
			dtrHosts = append(dtrHosts, h)
		}
	}
	return dtr.Cleanup(dtrHosts, swarmLeader)
}
