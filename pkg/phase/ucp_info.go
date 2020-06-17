package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	log "github.com/sirupsen/logrus"
)

// UcpInfo shows information about the UCP cluster
type UcpInfo struct{}

// Title for the phase
func (p *UcpInfo) Title() string {
	return "UCP cluster info"
}

// Run does the actual saving of the local state file
func (p *UcpInfo) Run(config *api.ClusterConfig) error {
	url := config.Spec.WebURL()
	log.Infof("Cluster is now configured. You can access your cluster admin UI at: %s", url)
	log.Infof("You can also download the admin client bundle with the following command: launchpad download-bundle --username <username> --password <password>")

	return nil
}
