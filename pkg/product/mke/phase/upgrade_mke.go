package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/pkg/phase"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/Mirantis/launchpad/pkg/swarm"
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

// Run the upgrade container.
func (p *UpgradeMKE) Run() error {
	leader := p.Config.Spec.SwarmLeader()

	p.EventProperties = map[string]interface{}{
		"upgraded": false,
	}
	if p.Config.Spec.MKE.Version == p.Config.Spec.MKE.Metadata.InstalledVersion {
		log.Infof("%s: cluster already at version %s, not running upgrade", leader, p.Config.Spec.MKE.Version)
		return nil
	}

	swarmClusterID := swarm.ClusterID(leader)
	upgradeFlags := p.Config.Spec.MKE.UpgradeFlags
	upgradeFlags.Merge(commonconfig.Flags{"--id", swarmClusterID})

	log.Debugf("%s: upgrade flags: %s", leader, upgradeFlags.Join())
	_, err := mke.Bootstrap("upgrade", *p.Config, mke.BootstrapOptions{OperationFlags: upgradeFlags, CleanupDisabled: p.CleanupDisabled(), ExecOptions: []exec.Option{exec.StreamOutput()}})
	if err != nil {
		return fmt.Errorf("%s: failed to run MKE upgrader: %w", leader, err)
	}

	originalInstalledVersion := p.Config.Spec.MKE.Metadata.InstalledVersion

	if err := mke.CollectFacts(leader, p.Config.Spec.MKE.Metadata); err != nil {
		return fmt.Errorf("%s: failed to collect existing MKE details: %w", leader, err)
	}

	p.EventProperties["upgraded"] = true
	p.EventProperties["installed_version"] = originalInstalledVersion
	p.EventProperties["upgraded_version"] = p.Config.Spec.MKE.Version

	return nil
}
