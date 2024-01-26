package phase

import (
	"fmt"

	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"

	"github.com/Mirantis/mcc/pkg/msr/msr2"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// UpgradeMSR is the phase implementation for running the actual msr upgrade container.
type UpgradeMSR struct {
	phase.Analytics
	phase.CleanupDisabling
	phase.BasicPhase
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

	err := p.Config.Spec.CheckMKEHealthRemote(h)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity: %w", h, err)
	}

	p.EventProperties = map[string]interface{}{
		"msr_upgraded": false,
	}

	if p.Config.Spec.MSR.Version == h.MSRMetadata.InstalledVersion {
		log.Infof("%s: MSR cluster already at version %s, not running upgrade", h, p.Config.Spec.MSR.Version)
		return nil
	}

	runFlags := common.Flags{"-i"}
	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}

	if h.Configurer.SELinuxEnabled(h) {
		runFlags.Add("--security-opt label=disable")
	}
	upgradeFlags := common.Flags{fmt.Sprintf("--existing-replica-id %s", h.MSRMetadata.MSR2.ReplicaID)}

	upgradeFlags.MergeOverwrite(msr2.BuildMKEFlags(p.Config))
	for _, f := range msr2.PluckSharedInstallFlags(p.Config.Spec.MSR.V2.InstallFlags, msr2.SharedInstallUpgradeFlags) {
		upgradeFlags.AddOrReplace(f)
	}
	upgradeFlags.MergeOverwrite(p.Config.Spec.MSR.V2.UpgradeFlags)

	upgradeCmd := h.Configurer.DockerCommandf("run %s %s upgrade %s", runFlags.Join(), p.Config.Spec.MSR.GetBootstrapperImage(), upgradeFlags.Join())
	log.Debugf("%s: Running MSR upgrade via bootstrapper", h)
	if err := h.Exec(upgradeCmd, exec.StreamOutput()); err != nil {
		return fmt.Errorf("%s: failed to run msr upgrade: %w", h, err)
	}

	msrMeta, err := msr2.CollectFacts(h)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing msr details: %w", h, err)
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if msrMeta.InstalledVersion != p.Config.Spec.MSR.Version {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return fmt.Errorf("%s: %w: upgraded msr version: %s does not match intended upgrade version: %s", h, errVersionMismatch, msrMeta.InstalledVersion, p.Config.Spec.MSR.Version)
	}

	p.EventProperties["msr_upgraded"] = true
	p.EventProperties["msr_installed_version"] = h.MSRMetadata.InstalledVersion
	p.EventProperties["msr_upgraded_version"] = p.Config.Spec.MSR.Version
	h.MSRMetadata = msrMeta

	return nil
}
