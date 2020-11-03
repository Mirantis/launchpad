package phase

import (
	log "github.com/sirupsen/logrus"
)

// Info shows information about the configured clusters
type Info struct {
	BasicPhase
}

// Title for the phase
func (p *Info) Title() string {
	return "UCP cluster info"
}

// Run ...
func (p *Info) Run() error {
	log.Info("Cluster is now configured.")

	ucpurl, err := p.config.Spec.UcpURL()
	if err == nil {
		log.Infof("UCP cluster admin UI: %s", ucpurl)
	}

	dtrurl, err := p.config.Spec.DtrURL()
	if err == nil {
		log.Infof("DTR cluster admin UI: %s", dtrurl)
	}

	log.Info("You can download the admin client bundle with the command 'launchpad client-config'")

	return nil
}
