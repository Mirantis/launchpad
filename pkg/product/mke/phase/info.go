package phase

import (
	"context"

	"github.com/Mirantis/mcc/pkg/mke"
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
			var msrURL string

			if p.Config.Spec.MSR.LoadBalancerURL != "" {
				msrURL = p.Config.Spec.MSR.LoadBalancerURL
			} else {
				kc, _, err := mke.KubeAndHelmFromConfig(p.Config)
				if err != nil {
					log.Debugf("failed to get msr URL: failed to get kube client: %s", err)
				}

				url, err := kc.MSRURL(context.Background(), p.Config.Spec.MSR.MSR3Config.Name)
				if err != nil {
					log.Debugf("failed to get msr URL: %s", err)
				} else {
					msrURL = url.String()
				}
			}

			log.Infof("MSR cluster admin UI: %s", msrURL)
		}
	}

	log.Info("You can download the admin client bundle with the command 'launchpad client-config'")

	return nil
}
