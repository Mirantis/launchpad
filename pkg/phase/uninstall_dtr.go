package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	"github.com/Mirantis/mcc/pkg/dtr"
	log "github.com/sirupsen/logrus"
)

// UninstallDTR is the phase implementation for running DTR uninstall
type UninstallDTR struct {
	Analytics
}

// Title prints the phase title
func (p *UninstallDTR) Title() string {
	return "Uninstall DTR components"
}

// Run an uninstall via CleanupDTRs
func (p *UninstallDTR) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()
	if !config.Spec.Dtr.Metadata.Installed {
		log.Infof("%s: DTR is not installed", swarmLeader.Address)
		return nil
	}

	var dtrHosts []*api.Host

	for _, h := range config.Spec.Hosts {
		if h.Role == "dtr" {
			dtrHosts = append(dtrHosts, h)
		}
	}
	return dtr.CleanupDtrs(dtrHosts, swarmLeader)
}
