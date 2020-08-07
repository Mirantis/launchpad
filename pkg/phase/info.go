package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	log "github.com/sirupsen/logrus"
)

// Info shows information about the configured clusters
type Info struct{}

// Title for the phase
func (p *Info) Title() string {
	return "UCP cluster info"
}

// Run does the actual saving of the local state file
func (p *Info) Run(config *api.ClusterConfig) error {
	urls := config.Spec.WebURLs()
	log.Info("Cluster is now configured.  You can access your admin UIs at:\n")
	log.Infof("UCP cluster admin UI: %s", urls.Ucp)
	// If the DTR URL is blank, it will still return a https://%s string, so
	// check for anything other than that
	if urls.Dtr != "https://" {
		log.Infof("DTR cluster admin UI: %s", urls.Dtr)
	}
	log.Infof("You can also download the admin client bundle with the following command: launchpad download-bundle --username <username> --password <password>")

	return nil
}
