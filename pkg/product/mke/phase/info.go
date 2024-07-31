package phase

import (
	"github.com/Mirantis/mcc/pkg/msr/msr3"
	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// Info shows information about the configured clusters.
type Info struct {
	phase.BasicPhase
}

// Title for the phase.
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

	if p.Config.Spec.MSR2 != nil {
		msrURL, err := p.Config.Spec.MSR2URL()
		if err == nil {
			log.Infof("MSR cluster admin UI: %s", msrURL.String())
		}
	}

	if p.Config.Spec.MSR3 != nil {
		msrURL, err := msr3.GetMSRURL(p.Config)
		if err != nil {
			log.Infof("failed to get MSR URL: %s", err)
		} else {
			log.Infof("MSR cluster admin UI: %s", msrURL)
		}
	}

	log.Info("You can download the admin client bundle with the command 'launchpad client-config'")

	return nil
}
