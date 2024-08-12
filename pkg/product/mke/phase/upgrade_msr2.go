package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr/msr2"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// UpgradeMSR is the phase implementation for running the actual msr upgrade container.
type UpgradeMSR2 struct {
	phase.Analytics
	phase.CleanupDisabling
	phase.BasicPhase
}

type msrUpgradeVersionNotExpectedError struct {
	host             *api.Host
	installedVersion string
	expectedVersion  string
}

func (e *msrUpgradeVersionNotExpectedError) Error() string {
	return fmt.Sprintf("%s: MSR2 upgrade version %s does not match expected version %s", e.host, e.installedVersion, e.expectedVersion)
}

// Title prints the phase title.
func (p *UpgradeMSR2) Title() string {
	return "Upgrade MSR2 components"
}

// ShouldRun should return true only when there is an upgrade to be performed.
func (p *UpgradeMSR2) ShouldRun() bool {
	if !p.Config.Spec.ContainsMSR2() {
		return false
	}
	h := p.Config.Spec.MSR2Leader()
	return p.Config.Spec.ContainsMSR() && h.MSR2Metadata != nil && h.MSR2Metadata.InstalledVersion != p.Config.Spec.MSR2.Version
}

// Run the upgrade container.
func (p *UpgradeMSR2) Run() error {
	h := p.Config.Spec.MSR2Leader()

	managers := p.Config.Spec.Managers()

	err := p.Config.Spec.CheckMKEHealthRemote(managers)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity: %w", h, err)
	}

	p.EventProperties = map[string]interface{}{
		"msr_upgraded": false,
	}

	if p.Config.Spec.MSR2.Version == h.MSR2Metadata.InstalledVersion {
		log.Infof("%s: MSR cluster already at version %s, not running upgrade", h, p.Config.Spec.MSR2.Version)
		return nil
	}

	runFlags := common.Flags{"-i"}
	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}

	if h.Configurer.SELinuxEnabled(h) {
		runFlags.Add("--security-opt label=disable")
	}
	upgradeFlags := common.Flags{fmt.Sprintf("--existing-replica-id %s", h.MSR2Metadata.ReplicaID)}

	upgradeFlags.MergeOverwrite(msr2.BuildMKEFlags(p.Config))
	for _, f := range msr2.PluckSharedInstallFlags(p.Config.Spec.MSR2.InstallFlags, msr2.SharedInstallUpgradeFlags) {
		upgradeFlags.AddOrReplace(f)
	}
	upgradeFlags.MergeOverwrite(p.Config.Spec.MSR2.UpgradeFlags)

	upgradeCmd := h.Configurer.DockerCommandf("run %s %s upgrade %s", runFlags.Join(), p.Config.Spec.MSR2.GetBootstrapperImage(), upgradeFlags.Join())
	log.Debugf("%s: Running MSR upgrade via bootstrapper", h)
	if err := h.Exec(upgradeCmd, exec.StreamOutput()); err != nil {
		return fmt.Errorf("%s: failed to run MSR upgrade: %w", h, err)
	}

	msrMeta, err := msr2.CollectFacts(h)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MSR details: %w", h, err)
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if msrMeta.InstalledVersion != p.Config.Spec.MSR2.Version {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return &msrUpgradeVersionNotExpectedError{host: h, installedVersion: msrMeta.InstalledVersion, expectedVersion: p.Config.Spec.MSR2.Version}
	}

	p.EventProperties["msr_upgraded"] = true
	p.EventProperties["msr_installed_version"] = h.MSR2Metadata.InstalledVersion
	p.EventProperties["msr_upgraded_version"] = p.Config.Spec.MSR2.Version
	h.MSR2Metadata = msrMeta

	return nil
}
