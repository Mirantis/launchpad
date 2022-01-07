package phase

import (
	"strings"

	"github.com/Mirantis/mcc/pkg/docker/hub"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

var Disable = false

// UpgradeCheck displays a notification of an upgrade being available
type UpgradeCheck struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title
func (p *UpgradeCheck) Title() string {
	return "Check For Upgrades"
}

func (p *UpgradeCheck) ShouldRun() bool {
	return !Disable
}

// Run the installer container
func (p *UpgradeCheck) Run() (err error) {
	mv, err := hub.LatestTag("mirantis", "ucp", strings.Contains(p.Config.Spec.MKE.Version, "-"))
	if err != nil {
		log.Errorf("failed to check for MKE upgrade: %s", err.Error())
	}

	mkeV, err := version.NewVersion(mv)
	if err != nil {
		log.Errorf("invalid MKE version response: %s", err.Error())
		return nil
	}

	mkeTargetV, err := version.NewVersion(p.Config.Spec.MKE.Version)
	if err != nil {
		log.Errorf("invalid MKE version in configuration: %s", err.Error())
		return err
	}

	if mkeV.GreaterThan(mkeTargetV) {
		log.Warnf("a newer version of MKE is available: %s (installing %s)", mv, mkeTargetV.String())
	}

	if !p.Config.Spec.ContainsMSR() {
		return nil
	}

	msrv, err := hub.LatestTag("mirantis", "dtr", strings.Contains(p.Config.Spec.MSR.Version, "-"))
	if err != nil {
		log.Errorf("failed to check for MSR upgrade: %s", err.Error())
		return nil
	}

	msrV, err := version.NewVersion(msrv)
	if err != nil {
		log.Errorf("invalid MSR version response: %s", err.Error())
		return nil
	}

	msrTargetV, err := version.NewVersion(p.Config.Spec.MSR.Version)
	if err != nil {
		log.Errorf("invalid MSR version in configuration: %s", err.Error())
		return err
	}

	if msrV.GreaterThan(msrTargetV) {
		log.Warnf("a newer version of MSR is available: %s (installing %s)", msrv, msrTargetV.String())
	}

	return nil
}
