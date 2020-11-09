package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"
	log "github.com/sirupsen/logrus"
)

// UpgradeUcp is the phase implementation for running the actual UCP upgrade container
type UpgradeUcp struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title
func (p *UpgradeUcp) Title() string {
	return "Upgrade UCP components"
}

// Run the installer container
func (p *UpgradeUcp) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	p.EventProperties = map[string]interface{}{
		"upgraded": false,
	}
	if p.Config.Spec.Ucp.Version == p.Config.Spec.Ucp.Metadata.InstalledVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", swarmLeader, p.Config.Spec.Ucp.Version)
		return nil
	}

	swarmClusterID := swarm.ClusterID(swarmLeader)
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run %s %s upgrade --id %s", strings.Join(runFlags, " "), p.Config.Spec.Ucp.GetBootstrapperImage(), swarmClusterID)
	log.Debugf("Running upgrade with cmd: %s", upgradeCmd)
	err := swarmLeader.Exec(upgradeCmd, exec.StreamOutput())
	if err != nil {
		return fmt.Errorf("failed to run UCP upgrade")
	}

	originalInstalledVersion := p.Config.Spec.Ucp.Metadata.InstalledVersion

	err = ucp.CollectFacts(swarmLeader, p.Config.Spec.Ucp.Metadata)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader, err.Error())
	}
	p.EventProperties["upgraded"] = true
	p.EventProperties["installed_version"] = originalInstalledVersion
	p.EventProperties["upgraded_version"] = p.Config.Spec.Ucp.Version

	return nil
}
