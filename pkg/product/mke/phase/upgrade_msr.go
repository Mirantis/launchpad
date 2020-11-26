package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"

	log "github.com/sirupsen/logrus"
)

// UpgradeMSR is the phase implementation for running the actual msr upgrade container
type UpgradeMSR struct {
	phase.Analytics
	MSRPhase
}

// Title prints the phase title
func (p *UpgradeMSR) Title() string {
	return "Upgrade MSR components"
}

// ShouldRun should return true only when there is an upgrade to be performed
func (p *UpgradeMSR) ShouldRun() bool {
	h := p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR() && h.MSRMetadata != nil && h.MSRMetadata.InstalledVersion != p.Config.Spec.MSR.Version
}

// Run the upgrade container
func (p *UpgradeMSR) Run() error {
	h := p.Config.Spec.MSRLeader()

	err := p.Config.Spec.CheckMKEHealthRemote(h)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity", h)
	}

	p.EventProperties = map[string]interface{}{
		"msr_upgraded": false,
	}

	if p.Config.Spec.MSR.Version == h.MSRMetadata.InstalledVersion {
		log.Infof("%s: msr cluster already at version %s, not running upgrade", h, p.Config.Spec.MSR.Version)
		return nil
	}

	runFlags := api.Flags{"--rm", "-i"}
	if h.Configurer.SELinuxEnabled() {
		runFlags.Add("--security-opt label=disable")
	}
	upgradeFlags := api.Flags{fmt.Sprintf("--existing-replica-id %s", h.MSRMetadata.ReplicaID)}

	upgradeFlags.MergeOverwrite(msr.BuildMKEFlags(p.Config))
	for _, f := range msr.PluckSharedInstallFlags(p.Config.Spec.MSR.InstallFlags, msr.SharedInstallUpgradeFlags) {
		upgradeFlags.AddOrReplace(f)
	}

	upgradeCmd := h.Configurer.DockerCommandf("run %s %s upgrade %s", runFlags.Join(), p.Config.Spec.MSR.GetBootstrapperImage(), upgradeFlags.Join())
	log.Debugf("%s: Running msr upgrade via bootstrapper", h)
	if err := h.Exec(upgradeCmd, exec.StreamOutput()); err != nil {
		return fmt.Errorf("%s: failed to run msr upgrade: %s", h, err.Error())
	}

	msrMeta, err := msr.CollectFacts(h)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing msr details: %s", h, err.Error())
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if msrMeta.InstalledVersion != p.Config.Spec.MSR.Version {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return fmt.Errorf("%s: upgraded msr version: %s does not match intended upgrade version: %s", h, msrMeta.InstalledVersion, p.Config.Spec.MSR.Version)
	}

	p.EventProperties["msr_upgraded"] = true
	p.EventProperties["msr_installed_version"] = h.MSRMetadata.InstalledVersion
	p.EventProperties["msr_upgraded_version"] = p.Config.Spec.MSR.Version
	h.MSRMetadata = msrMeta

	return nil
}
