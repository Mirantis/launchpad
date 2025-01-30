package phase

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

// ValidateFacts phase implementation to validate facts from config and collected metadata.
type ValidateFacts struct {
	phase.Analytics
	phase.BasicPhase
	Force bool
}

// Title for the phase.
func (p *ValidateFacts) Title() string {
	return "Validate Facts"
}

// Run validate configuration facts.
func (p *ValidateFacts) Run() error {
	if !p.Config.Spec.MKE.InstallFlags.Include("--san") {
		p.populateSan()
	}

	_ = p.Config.Spec.Hosts.Each(func(h *api.Host) error {
		if h.Configurer != nil && h.Configurer.SELinuxEnabled(h) {
			h.DaemonConfig["selinux-enabled"] = true
			log.Infof("%s: adding 'selinux-enabled=true' to host container runtime config", h)
		}

		return nil
	})

	if err := p.validateMKEVersionJump(); err != nil {
		if p.Force {
			log.Warnf("%s: continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	if err := p.validateMSRVersionJump(); err != nil {
		if p.Force {
			log.Warnf("%s: continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	if err := p.validateDataPlane(); err != nil {
		if p.Force {
			log.Warnf("%s: continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	return nil
}

func (p *ValidateFacts) populateSan() {
	mgrs := p.Config.Spec.Managers()
	for _, h := range mgrs {
		f := fmt.Sprintf("--san=%s", h.Address())
		p.Config.Spec.MKE.InstallFlags.Add(f)
		log.Warnf("%s: added manager node's public address to mke installFlag SANs: %s", h, f)
	}
}

var errInvalidUpgradePath = errors.New("invalid upgrade path")

// validateMSRVersionJump validates MKE upgrade path.
func (p *ValidateFacts) validateMKEVersionJump() error {
	if p.Config.Spec.MKE.Metadata.Installed && p.Config.Spec.MKE.Metadata.InstalledVersion != "" {
		installedMKE, err := version.NewVersion(p.Config.Spec.MKE.Metadata.InstalledVersion)
		if err != nil {
			return fmt.Errorf("can't parse installed MKE version: %w", err)
		}
		targetMKE, err := version.NewVersion(p.Config.Spec.MKE.Version)
		if err != nil {
			return fmt.Errorf("can't parse target MKE version: %w", err)
		}

		if mke.VersionGreaterThan(installedMKE, targetMKE) {
			return fmt.Errorf("%w: can't downgrade MKE %s to %s", errInvalidUpgradePath, installedMKE, targetMKE)
		}

		installedSegments := installedMKE.Segments()
		targetSegments := targetMKE.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("%w: can't upgrade MKE directly from %s to %s - need to upgrade to %d.%d first", errInvalidUpgradePath, installedMKE, targetMKE, installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}

// validateMSRVersionJump validates MSR upgrade path.
func (p *ValidateFacts) validateMSRVersionJump() error {
	msrLeader := p.Config.Spec.MSRLeader()
	if p.Config.Spec.MSR != nil && msrLeader.MSRMetadata != nil && msrLeader.MSRMetadata.Installed && msrLeader.MSRMetadata.InstalledVersion != "" {
		installedMSR, err := version.NewVersion(msrLeader.MSRMetadata.InstalledVersion)
		if err != nil {
			return fmt.Errorf("can't parse installed MSR version: %w", err)
		}
		targetMSR, err := version.NewVersion(p.Config.Spec.MSR.Version)
		if err != nil {
			return fmt.Errorf("can't parse target MSR version: %w", err)
		}

		if mke.VersionGreaterThan(installedMSR, targetMSR) {
			return fmt.Errorf("%w: can't downgrade MSR %s to %s", errInvalidUpgradePath, installedMSR, targetMSR)
		}

		installedSegments := installedMSR.Segments()
		targetSegments := targetMSR.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("%w: can't upgrade MSR directly from %s to %s - need to upgrade to %d.%d first", errInvalidUpgradePath, installedMSR, targetMSR, installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}

var errInvalidDataPlane = errors.New("invalid data plane settings")

// validateDataPlane checks if the calico data plane would get changed (VXLAN <-> VPIP).
func (p *ValidateFacts) validateDataPlane() error {
	log.Debug("validating data plane settings")

	idx := p.Config.Spec.MKE.InstallFlags.Index("--calico-vxlan")
	if idx < 0 {
		return nil
	}

	val := p.Config.Spec.MKE.InstallFlags.GetValue("--calico-vxlan")
	var valB bool
	if val == "" {
		valB = true
	} else {
		v, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("can't parse --calico-vxlan value: %w", err)
		}
		valB = v
	}

	// User has explicitly defined --calico-vxlan=false but there is a windows host in the config
	if !valB {
		if p.Config.Spec.Hosts.Include(func(h *api.Host) bool { return h.IsWindows() }) {
			return fmt.Errorf("%w: calico IPIP can't be used on Windows", errInvalidDataPlane)
		}

		log.Debug("no windows hosts found")
	}

	if !p.Config.Spec.MKE.Metadata.Installed {
		log.Debug("no existing MKE installation")
		return nil
	}

	// User has explicitly defined --calico-vxlan=false but there is already a calico with vxlan
	if p.Config.Spec.MKE.Metadata.VXLAN {
		log.Debug("mke has been installed with calico + vxlan")
		if !valB {
			return fmt.Errorf("%w: calico configured with VXLAN, can't automatically change to IPIP", errInvalidDataPlane)
		}
	} else {
		log.Debug("mke has been installed with calico + vpip")
		// User has explicitly defined --calico-vxlan=true but there is already a calico with ipip
		if valB {
			return fmt.Errorf("%w: calico configured with IPIP, can't automatically change to VXLAN", errInvalidDataPlane)
		}
	}

	log.Debug("data plane settings check passed")

	return nil
}
