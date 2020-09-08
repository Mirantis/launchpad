package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/dtr"
	log "github.com/sirupsen/logrus"
)

// UpgradeDtr is the phase implementation for running the actual DTR upgrade container
type UpgradeDtr struct {
	Analytics
}

// Title prints the phase title
func (p *UpgradeDtr) Title() string {
	return "Upgrade DTR components"
}

// Run the upgrade container
func (p *UpgradeDtr) Run(config *api.ClusterConfig) error {
	dtrLeader := config.Spec.DtrLeader()

	p.EventProperties = map[string]interface{}{
		"dtr_upgraded": false,
	}

	// Check specified bootstrapper images version
	bootstrapperVersion, err := dtr.GetBootstrapVersion(dtrLeader, config)
	if err != nil {
		return NewError("Failed to check DTR bootstrapper image version")
	}
	installedVersion := config.Spec.Dtr.Metadata.InstalledVersion
	if bootstrapperVersion == installedVersion {
		log.Infof("%s: DTR cluster already at version %s, not running upgrade", dtrLeader.Address, bootstrapperVersion)
		return nil
	}
	log.Debugf("Proceeding with DTR upgrade: bootstrapperVersion: %s does not match installedVersion: %s", bootstrapperVersion, installedVersion)

	runFlags := []string{"--rm", "-i"}
	if dtrLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	upgradeFlags := []string{
		fmt.Sprintf("--existing-replica-id %s", config.Spec.Dtr.Metadata.DtrLeaderReplicaID),
	}
	ucpFlags := dtr.BuildUcpFlags(config)
	upgradeFlags = append(upgradeFlags, ucpFlags...)
	for _, f := range dtr.PluckSharedInstallFlags(config.Spec.Dtr.InstallFlags, dtr.SharedInstallUpgradeFlags) {
		upgradeFlags = append(upgradeFlags, f)
	}

	upgradeCmd := dtrLeader.Configurer.DockerCommandf("run %s %s upgrade %s", strings.Join(runFlags, " "), config.Spec.Dtr.GetBootstrapperImage(), strings.Join(upgradeFlags, " "))
	log.Debug("Running DTR upgrade via bootstrapper")
	err = dtrLeader.ExecCmd(upgradeCmd, "", true, false)
	if err != nil {
		return NewError("Failed to run DTR upgrade")
	}

	dtrMeta, err := dtr.CollectDtrFacts(dtrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing DTR details: %s", dtrLeader.Address, err.Error())
	}

	// Check to make sure installedversion matches bootstrapperVersion
	if dtrMeta.InstalledVersion != bootstrapperVersion {
		// If our newly collected facts do not match the version we upgraded to
		// then the upgrade has failed
		return NewError(fmt.Sprintf("Upgraded DTR version: %s does not match intended upgrade version: %s", dtrMeta.InstalledVersion, bootstrapperVersion))
	}

	p.EventProperties["dtr_upgraded"] = true
	p.EventProperties["dtr_installed_version"] = installedVersion
	p.EventProperties["dtr_upgraded_version"] = bootstrapperVersion
	config.Spec.Dtr.Metadata = dtrMeta

	return nil
}
