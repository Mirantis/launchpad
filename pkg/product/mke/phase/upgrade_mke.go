package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/hashicorp/go-version"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// UpgradeMKE is the phase implementation for running the actual MKE upgrade container
type UpgradeMKE struct {
	phase.Analytics
	phase.BasicPhase
	phase.CleanupDisabling
}

// Title prints the phase title
func (p *UpgradeMKE) Title() string {
	return "Upgrade MKE components"
}

// Run the upgrade container
func (p *UpgradeMKE) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	p.EventProperties = map[string]interface{}{
		"upgraded": false,
	}
	if p.Config.Spec.MKE.Version == p.Config.Spec.MKE.Metadata.InstalledVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", swarmLeader, p.Config.Spec.MKE.Version)
		return nil
	}

	if p.Config.Spec.MKE.Metadata.VersionDefaulted {
		mkev, err := version.NewVersion(p.Config.Spec.MKE.Version)
		if err != nil {
			return err
		}

		instv, err := version.NewVersion(p.Config.Spec.MKE.Metadata.InstalledVersion)
		if err != nil {
			return err
		}

		if mkev.GreaterThan(instv) {
			log.Warnf("%s: a newer version of MKE is available, set configuration spec.mke.version %s to upgrade", swarmLeader, mkev.String())
		}
	}

	swarmClusterID := swarm.ClusterID(swarmLeader)
	runFlags := common.Flags{"-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}

	if swarmLeader.Configurer.SELinuxEnabled(swarmLeader) {
		runFlags.Add("--security-opt label=disable")
	}
	upgradeCmd := swarmLeader.Configurer.DockerCommandf("run %s %s upgrade --id %s %s", runFlags.Join(), p.Config.Spec.MKE.GetBootstrapperImage(), swarmClusterID, p.Config.Spec.MKE.UpgradeFlags.Join())
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
