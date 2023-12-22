package phase

import (
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

	if p.Config.Spec.MSR != nil {
		switch p.Config.Spec.MSR.MajorVersion() {
		case 2:
			msrURL, err := p.Config.Spec.MSR2URL()
			if err == nil {
				log.Infof("MSR cluster admin UI: %s", msrURL.String())
			}
		case 3:
			msrURL, err := getMSRURL(p.Config)
			if err != nil {
				log.Infof("failed to get msr URL: %s", err)
			} else {
				log.Infof("MSR cluster admin UI: %s", msrURL)
			}
		}
	}

	log.Info("You can download the admin client bundle with the command 'launchpad client-config'")

	return nil
}
