package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/docker/hub"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

// DisableUpgradeCheck for offline use.
var DisableUpgradeCheck = false

// UpgradeCheck displays a notification of an upgrade being available.
type UpgradeCheck struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title.
func (p *UpgradeCheck) Title() string {
	return "Check For Upgrades"
}

// ShouldRun will return false when upgrades checks are disabled.
func (p *UpgradeCheck) ShouldRun() bool {
	return !DisableUpgradeCheck
}

// Run the installer container.
func (p *UpgradeCheck) Run() (err error) {
	if p.Config.Spec.ContainsMKE() {
		mkeTag, err := hub.LatestTag(hub.RegistryDockerHub, "mirantis", "ucp", strings.Contains(p.Config.Spec.MKE.Version, "-"))
		if err != nil {
			log.Errorf("failed to check for MKE upgrade: %v", err)
			return nil
		}

		mkeV, err := version.NewVersion(mkeTag)
		if err != nil {
			log.Errorf("invalid MKE version response: %s", err.Error())
			return nil
		}

		mkeTargetV, err := version.NewVersion(p.Config.Spec.MKE.Version)
		if err != nil {
			log.Errorf("invalid MKE version in configuration: %s", err.Error())
			return fmt.Errorf("invalid MKE version in configuration: %w", err)
		}

		if mkeV.GreaterThan(mkeTargetV) {
			log.Warnf("a newer version of MKE is available: %s (installing %s)", mkeTag, mkeTargetV.String())
		}
	}

	if p.Config.Spec.ContainsMSR2() {
		msrv, err := hub.LatestTag(hub.RegistryDockerHub, "mirantis", "dtr", strings.Contains(p.Config.Spec.MSR2.Version, "-"))
		if err != nil {
			log.Errorf("failed to check for MSR2 upgrade: %s", err.Error())
			return nil
		}

		msrV, err := version.NewVersion(msrv)
		if err != nil {
			log.Errorf("invalid MSR2 version response: %s", err.Error())
			return nil
		}

		msrTargetV, err := version.NewVersion(p.Config.Spec.MSR2.Version)
		if err != nil {
			log.Errorf("invalid MSR2 version in configuration: %s", err.Error())
			return fmt.Errorf("invalid MSR2 version in configuration: %w", err)
		}

		if msrV.GreaterThan(msrTargetV) {
			log.Warnf("a newer version of MSR2 is available: %s (installing %s)", msrv, msrTargetV.String())
		}
	}

	if p.Config.Spec.ContainsMSR3() {
		msrv, err := hub.LatestTag(hub.RegistryMirantis, "msr", "msr-api", strings.Contains(p.Config.Spec.MSR3.Version, "-"))
		if err != nil {
			log.Errorf("failed to check for MSR3 upgrade: %s", err.Error())
			return nil
		}

		msrV, err := version.NewVersion(msrv)
		if err != nil {
			log.Errorf("invalid MSR3 version response: %s", err.Error())
			return nil
		}

		msrTargetV, err := version.NewVersion(p.Config.Spec.MSR2.Version)
		if err != nil {
			log.Errorf("invalid MSR3 version in configuration: %s", err.Error())
			return fmt.Errorf("invalid MSR3 version in configuration: %w", err)
		}

		if msrV.GreaterThan(msrTargetV) {
			log.Warnf("a newer version of MSR3 is available: %s (installing %s)", msrv, msrTargetV.String())
		}
	}

	return nil
}
