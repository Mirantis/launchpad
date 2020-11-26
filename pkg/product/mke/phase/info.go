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
	return "MKE cluster info"
}

// Run ...
func (p *Info) Run() error {
	log.Info("Cluster is now configured.")

	mkeurl, err := p.Config.Spec.MKEURL()
	if err == nil {
		log.Infof("MKE cluster admin UI: %s", mkeurl)
	}

	msrurl, err := p.Config.Spec.MSRURL()
	if err == nil {
		log.Infof("MSR cluster admin UI: %s", msrurl)
	}

	log.Info("You can download the admin client bundle with the command 'launchpad client-config'")

	return nil
}
