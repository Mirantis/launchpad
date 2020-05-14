package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// InstallUCP is the phase implementation for running the actual UCP installer container
type InstallUCP struct{}

// Title prints the phase title
func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

// Run the installer container
func (p *InstallUCP) Run(config *config.ClusterConfig) *PhaseError {
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
		return NewPhaseError("Failed to run UCP installer")
	}
	return nil
}
