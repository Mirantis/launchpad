package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// UpgradeMKE is the phase implementation for running the actual MKE upgrade container
type UpgradeMKE struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title
func (p *UpgradeMKE) Title() string {
	return "Upgrade MKE components"
}

// Run the installer container
func (p *UpgradeMKE) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	p.EventProperties = map[string]interface{}{
		"upgraded": false,
	}
	if p.Config.Spec.MKE.Version == p.Config.Spec.MKE.Metadata.InstalledVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", swarmLeader, p.Config.Spec.MKE.Version)
		return nil
	}

	swarmClusterID := swarm.ClusterID(swarmLeader)
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run %s %s upgrade --id %s", strings.Join(runFlags, " "), p.Config.Spec.MKE.GetBootstrapperImage(), swarmClusterID)
	log.Debugf("Running upgrade with cmd: %s", upgradeCmd)
	err := swarmLeader.Exec(upgradeCmd, exec.StreamOutput())
	if err != nil {
		return fmt.Errorf("failed to run MKE upgrade")
	}

	originalInstalledVersion := p.Config.Spec.MKE.Metadata.InstalledVersion

	err = mke.CollectFacts(swarmLeader, p.Config.Spec.MKE.Metadata)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MKE details: %s", swarmLeader, err.Error())
	}
	p.EventProperties["upgraded"] = true
	p.EventProperties["installed_version"] = originalInstalledVersion
	p.EventProperties["upgraded_version"] = p.Config.Spec.MKE.Version

	return nil
}
