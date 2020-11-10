package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// Info shows information about the configured clusters
type Info struct {
	phase.BasicPhase
}

// Title for the phase
func (p *Info) Title() string {
	return "UCP cluster info"
}

// Run ...
func (p *Info) Run() error {
	log.Info("Cluster is now configured.")

	ucpurl, err := p.Config.Spec.UcpURL()
	if err == nil {
		log.Infof("UCP cluster admin UI: %s", ucpurl)
	}

	dtrurl, err := p.Config.Spec.DtrURL()
	if err == nil {
		log.Infof("DTR cluster admin UI: %s", dtrurl)
	}

	log.Info("You can download the admin client bundle with the command 'launchpad client-config'")

	return nil
}
