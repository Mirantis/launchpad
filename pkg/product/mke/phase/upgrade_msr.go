package phase

import (
	"fmt"
	"strings"

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

// Run the upgrade container
func (p *UpgradeMSR) Run() error {
	msrLeader := p.Config.Spec.MSRLeader()

	err := p.Config.Spec.CheckMKEHealthRemote(msrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity", msrLeader)
	}

	p.EventProperties = map[string]interface{}{
		"msr_upgraded": false,
	}

	if p.Config.Spec.MSR.Version == p.Config.Spec.MSR.Metadata.InstalledVersion {
		log.Infof("%s: msr cluster already at version %s, not running upgrade", msrLeader, p.Config.Spec.MSR.Version)
		return nil
	}

	runFlags := []string{"--rm", "-i"}
	if msrLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	upgradeFlags := []string{
		fmt.Sprintf("--existing-replica-id %s", p.Config.Spec.MSR.Metadata.MSRLeaderReplicaID),
	}
	mkeFlags := msr.BuildMKEFlags(p.Config)
	upgradeFlags = append(upgradeFlags, mkeFlags...)
	for _, f := range msr.PluckSharedInstallFlags(p.Config.Spec.MSR.InstallFlags, msr.SharedInstallUpgradeFlags) {
		upgradeFlags = append(upgradeFlags, f)
	}

	upgradeCmd := msrLeader.Configurer.DockerCommandf("run %s %s upgrade %s", strings.Join(runFlags, " "), p.Config.Spec.MSR.GetBootstrapperImage(), strings.Join(upgradeFlags, " "))
	log.Debug("Running msr upgrade via bootstrapper")
	err = msrLeader.Exec(upgradeCmd, exec.StreamOutput())
	if err != nil {
		return fmt.Errorf("failed to run msr upgrade: %s", err.Error())
	}

	msrMeta, err := msr.CollectFacts(msrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing msr details: %s", msrLeader, err.Error())
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if msrMeta.InstalledVersion != p.Config.Spec.MSR.Version {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return fmt.Errorf("%s: upgraded msr version: %s does not match intended upgrade version: %s", msrLeader, msrMeta.InstalledVersion, p.Config.Spec.MSR.Version)
	}

	p.EventProperties["msr_upgraded"] = true
	p.EventProperties["msr_installed_version"] = p.Config.Spec.MSR.Metadata.InstalledVersion
	p.EventProperties["msr_upgraded_version"] = p.Config.Spec.MSR.Version
	p.Config.Spec.MSR.Metadata = msrMeta

	return nil
}
