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
func (p *InstallUCP) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]
	ucpExists, existingVersion, err := ucpExists(swarmLeader)
	if err != nil {
		return fmt.Errorf("Failed to check existing UCP version")
	}

	if ucpExists {
		log.Warnf("Existing UCP detected at version %s. Upgrades not supported, yet!", existingVersion)
		return nil // To let the rest of the process continue, e.g. join new workers etc.
	}

	image := fmt.Sprintf("%s/ucp:%s", config.Ucp.ImageRepo, config.Ucp.Version)
	flags := strings.Join(config.Ucp.InstallFlags, " ")
	installCmd := fmt.Sprintf("sudo docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install %s", image, flags)
	log.Debugf("Running installer with cmd: %s", installCmd)
	err = swarmLeader.Exec(installCmd)
	if err != nil {
		return fmt.Errorf("Failed to run UCP installer")
	}
	return nil
}

// checks whether UCP is already running. If it is also returns the current version.
func ucpExists(swarmLeader *config.Host) (bool, string, error) {
	output, err := swarmLeader.ExecWithOutput(`sudo docker inspect --format '{{ index .Config.Labels "com.docker.ucp.version"}}' ucp-proxy`)
	if err != nil {
		// We need to check the output to check if the container does not exist
		if strings.Contains(output, "No such object") {
			return false, "", nil
		}
		return false, "", err
	}
	return true, output, nil
}
