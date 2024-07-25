package phase

import (
	"fmt"

	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// UpgradeMKE is the phase implementation for running the actual MKE upgrade container.
type UpgradeMKE struct {
	phase.Analytics
	phase.BasicPhase
	phase.CleanupDisabling
}

// Title prints the phase title.
func (p *UpgradeMKE) Title() string {
	return "Upgrade MKE components"
}

// ShouldRun should return true only when there is an installation to be
// performed.
func (p *UpgradeMKE) ShouldRun() bool {
	return p.Config.Spec.MKE != nil
}

// Run the upgrade container.
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
	runFlags := common.Flags{"-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}
	upgradeFlags := p.Config.Spec.MKE.UpgradeFlags
	upgradeFlags.Merge(common.Flags{"--id", swarmClusterID})

	if mcclog.Debug {
		upgradeFlags.AddUnlessExist("--debug")
	}

	if swarmLeader.Configurer.SELinuxEnabled(swarmLeader) {
		runFlags.Add("--security-opt label=disable")
	}

	log.Debugf("%s: upgrade flags: %s", swarmLeader, upgradeFlags.Join())
	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run %s %s upgrade %s", runFlags.Join(), p.Config.Spec.MKE.GetBootstrapperImage(), upgradeFlags.Join())
	err := swarmLeader.Exec(upgradeCmd, exec.StreamOutput())
	if err != nil {
		return fmt.Errorf("%s: failed to run MKE upgrader: %w", swarmLeader, err)
	}

	originalInstalledVersion := p.Config.Spec.MKE.Metadata.InstalledVersion

	if err := mke.CollectFacts(swarmLeader, p.Config.Spec.MKE.Metadata); err != nil {
		return fmt.Errorf("%s: failed to collect existing MKE details: %w", swarmLeader, err)
	}

	p.EventProperties["upgraded"] = true
	p.EventProperties["installed_version"] = originalInstalledVersion
	p.EventProperties["upgraded_version"] = p.Config.Spec.MKE.Version

	return nil
}
