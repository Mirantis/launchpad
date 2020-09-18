package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"
	log "github.com/sirupsen/logrus"
)

// UpgradeUcp is the phase implementation for running the actual UCP upgrade container
type UpgradeUcp struct {
	Analytics
	BasicPhase
}

// Title prints the phase title
func (p *UpgradeUcp) Title() string {
	return "Upgrade UCP components"
}

// Run the installer container
func (p *UpgradeUcp) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()

	p.EventProperties = map[string]interface{}{
		"upgraded": false,
	}
	// Check specified bootstrapper images version
	bootstrapperVersion, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`image inspect %s --format '{{ index .Config.Labels "com.docker.ucp.version"}}'`, p.config.Spec.Ucp.GetBootstrapperImage()))
	if err != nil {
		return NewError("Failed to check bootstrapper image version")
	}
	installedVersion := p.config.Spec.Ucp.Metadata.InstalledVersion
	if bootstrapperVersion == installedVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", swarmLeader.Address, bootstrapperVersion)
		return nil
	}

	swarmClusterID := swarm.ClusterID(swarmLeader)
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run %s %s upgrade --id %s", strings.Join(runFlags, " "), p.config.Spec.Ucp.GetBootstrapperImage(), swarmClusterID)
	log.Debugf("Running upgrade with cmd: %s", upgradeCmd)
	err = swarmLeader.ExecCmd(upgradeCmd, "", true, false)
	if err != nil {
		return NewError("Failed to run UCP upgrade")
	}

	err = ucp.CollectUcpFacts(swarmLeader, p.config.Spec.Ucp.Metadata)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	p.EventProperties["upgraded"] = true
	p.EventProperties["installed_version"] = installedVersion
	p.EventProperties["upgraded_version"] = bootstrapperVersion

	return nil
}
