package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/Mirantis/launchpad/pkg/msr"
	"github.com/Mirantis/launchpad/pkg/phase"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	log "github.com/sirupsen/logrus"
)

// UpgradeMSR is the phase implementation for running the actual msr upgrade container.
type UpgradeMSR struct {
	phase.Analytics
	phase.CleanupDisabling
	MSRPhase
}

// Title prints the phase title.
func (p *UpgradeMSR) Title() string {
	return "Upgrade MSR components"
}

// ShouldRun should return true only when there is an upgrade to be performed.
func (p *UpgradeMSR) ShouldRun() bool {
	h := p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR() && h.MSRMetadata != nil && h.MSRMetadata.InstalledVersion != p.Config.Spec.MSR.Version
}

// Run the upgrade container.
func (p *UpgradeMSR) Run() error {
	h := p.Config.Spec.MSRLeader()

	p.EventProperties = map[string]interface{}{
		"msr_upgraded": false,
	}

	if p.Config.Spec.MSR.Version == h.MSRMetadata.InstalledVersion {
		log.Infof("%s: msr cluster already at version %s, not running upgrade", h, p.Config.Spec.MSR.Version)
		return nil
	}

	upgradeFlags := common.Flags{fmt.Sprintf("--existing-replica-id %s", h.MSRMetadata.ReplicaID)}

	upgradeFlags.MergeOverwrite(msr.BuildMKEFlags(p.Config))
	for _, f := range msr.PluckSharedInstallFlags(p.Config.Spec.MSR.InstallFlags, msr.SharedInstallUpgradeFlags) {
		upgradeFlags.AddOrReplace(f)
	}
	upgradeFlags.MergeOverwrite(p.Config.Spec.MSR.UpgradeFlags)

	if _, err := msr.Bootstrap("upgrade", *p.Config, msr.BootstrapOptions{OperationFlags: upgradeFlags, CleanupDisabled: p.CleanupDisabled()}); err != nil {
		return fmt.Errorf("%s: failed to run MSR upgrader: %w", h, err)
	}

	msrMeta, err := msr.CollectFacts(h)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing msr details: %w", h, err)
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if msrMeta.InstalledVersion != p.Config.Spec.MSR.Version {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return fmt.Errorf("%s: %w: upgraded msr version: %s does not match intended upgrade version: %s", h, constant.ErrVersionMismatch, msrMeta.InstalledVersion, p.Config.Spec.MSR.Version)
	}

	p.EventProperties["msr_upgraded"] = true
	p.EventProperties["msr_installed_version"] = h.MSRMetadata.InstalledVersion
	p.EventProperties["msr_upgraded_version"] = p.Config.Spec.MSR.Version
	h.MSRMetadata = msrMeta

	return nil
}
