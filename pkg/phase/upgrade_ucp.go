package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"

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
	swarmLeader := config.Managers()[0]

	// Check specified bootstrapper images version
	bootstrapperVersion, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`image inspect %s --format '{{ index .Config.Labels "com.docker.ucp.version"}}'`, config.Ucp.GetBootstrapperImage()))
	if err != nil {
		return NewError("Failed to check bootstrapper image version")
	}

	if bootstrapperVersion == config.Ucp.Metadata.InstalledVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", swarmLeader.Address, bootstrapperVersion)
		return nil
	}

	swarmClusterID := swarm.ClusterID(swarmLeader)

	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s upgrade --id %s", config.Ucp.GetBootstrapperImage(), swarmClusterID)
	log.Debugf("Running upgrade with cmd: %s", upgradeCmd)
	err = swarmLeader.ExecCmd(upgradeCmd, "", true)
	if err != nil {
		return NewError("Failed to run UCP upgrade")
	}

	ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	config.Ucp.Metadata = ucpMeta

	return nil
}
