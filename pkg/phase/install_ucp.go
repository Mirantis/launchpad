package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// InstallUCP is the phase implementation for running the actual UCP installer container
type InstallUCP struct{}

// Title prints the phase title
func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

// Run the installer container
func (p *InstallUCP) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]

	if config.Ucp.Metadata.Installed {
		log.Infof("UCP already installed at version %s, not running installer", config.Ucp.Metadata.InstalledVersion)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", config.Ucp.ImageRepo, config.Ucp.Version)
	flags := strings.Join(config.Ucp.InstallFlags, " ")
	installCmd := fmt.Sprintf("sudo docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install %s", image, flags)
	err := swarmLeader.Exec(installCmd)
	if err != nil {
		return NewError("Failed to run UCP installer")
	}

	ucpMeta, err := util.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	config.Ucp.Metadata = ucpMeta

	return nil
}
