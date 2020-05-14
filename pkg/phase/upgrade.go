package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/config"

	log "github.com/sirupsen/logrus"
)

// UpgradeUcp is the phase implementation for running the actual UCP upgrade container
type UpgradeUcp struct{}

// Title prints the phase title
func (p *UpgradeUcp) Title() string {
	return "Upgrade UCP components"
}

// Run the installer container
func (p *UpgradeUcp) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]

	// Check specified bootstrapper images version
	bootstrapperVersion, err := swarmLeader.ExecWithOutput(fmt.Sprintf(`sudo docker image inspect %s --format '{{ index .Config.Labels "com.docker.ucp.version"}}'`, config.Ucp.GetBootstrapperImage()))
	if err != nil {
		return NewError("Failed to check bootstrapper image version")
	}

	if bootstrapperVersion == config.Ucp.Metadata.InstalledVersion {
		log.Infof("Cluster already at version %s, not running upgrade", bootstrapperVersion)
		return nil
	}

	swarmClusterID := util.SwarmClusterID(swarmLeader)

	upgradeCmd := fmt.Sprintf("sudo docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s upgrade --id %s", config.Ucp.GetBootstrapperImage(), swarmClusterID)
	log.Debugf("Running upgrade with cmd: %s", upgradeCmd)
	err = swarmLeader.Exec(upgradeCmd)
	if err != nil {
		return NewError("Failed to run UCP upgrade")
	}

	return nil
}
