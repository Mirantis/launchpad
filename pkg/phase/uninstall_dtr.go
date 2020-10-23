package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/dtr"
	log "github.com/sirupsen/logrus"
)

// UninstallDTR is the phase implementation for running DTR uninstall
type UninstallDTR struct {
	Analytics
	BasicPhase
}

// Title prints the phase title
func (p *UninstallDTR) Title() string {
	return "Uninstall DTR components"
}

// Run an uninstall via dtr.Cleanup
func (p *UninstallDTR) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()
	if !p.config.Spec.Dtr.Metadata.Installed {
		log.Infof("%s: DTR is not installed", swarmLeader.Address)
		return nil
	}

	var dtrHosts []*api.Host

	for _, h := range p.config.Spec.Hosts {
		if h.Role == "dtr" {
			dtrHosts = append(dtrHosts, h)
		}
	}
	return dtr.Cleanup(dtrHosts, swarmLeader)
}
