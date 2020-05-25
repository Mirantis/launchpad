package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"

	log "github.com/sirupsen/logrus"
)

// UpgradeUcp is the phase implementation for running the actual UCP upgrade container
type UpgradeUcp struct{}

// Title prints the phase title
func (p *UpgradeUcp) Title() string {
	return "Upgrade UCP components"
}

// Run the installer container
func (p *UpgradeUcp) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	// Check specified bootstrapper images version
	bootstrapperVersion, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`image inspect %s --format '{{ index .Config.Labels "com.docker.ucp.version"}}'`, config.Spec.Ucp.GetBootstrapperImage()))
	if err != nil {
		return NewError("Failed to check bootstrapper image version")
	}

	if bootstrapperVersion == config.Spec.Ucp.Metadata.InstalledVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", swarmLeader.Address, bootstrapperVersion)
		return nil
	}

	swarmClusterID := swarm.ClusterID(swarmLeader)

	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s upgrade --id %s", config.Spec.Ucp.GetBootstrapperImage(), swarmClusterID)
	log.Debugf("Running upgrade with cmd: %s", upgradeCmd)
	err = swarmLeader.Exec(upgradeCmd, exec.StreamOutput())
	if err != nil {
		return NewError("Failed to run UCP upgrade")
	}

	ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	config.Spec.Ucp.Metadata = ucpMeta

	return nil
}
