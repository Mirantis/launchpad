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

// Run does the actual saving of the local state file
func (p *Info) Run() error {
	urls := p.config.Spec.WebURLs()
	log.Info("Cluster is now configured.  You can access your admin UIs at:\n")
	log.Infof("UCP cluster admin UI: %s", urls.Ucp)
	if urls.Dtr != "" {
		log.Infof("DTR cluster admin UI: %s", urls.Dtr)
	}
	log.Infof("You can also download the admin client bundle with the following command: launchpad download-bundle --username <username> --password <password>")

	return nil
}
