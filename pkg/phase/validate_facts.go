package phase

import (
	"fmt"
	"strconv"

	"github.com/Mirantis/mcc/pkg/api"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

// ValidateFacts phase implementation to validate facts from config and collected metadata
type ValidateFacts struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *ValidateFacts) Title() string {
	return "Validate Facts"
}

// Run validate configuration facts
func (p *ValidateFacts) Run() error {
	if !p.config.Spec.Ucp.InstallFlags.Include("--san") {
		p.populateSan()
	}

	if err := p.validateUCPVersionJump(p.config); err != nil {
		if p.config.Spec.Metadata.Force {
			log.Warnf("%s - continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	if err := p.validateDTRVersionJump(p.config); err != nil {
		if p.config.Spec.Metadata.Force {
			log.Warnf("%s - continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	if err := p.validateDataPlane(p.config); err != nil {
		if p.config.Spec.Metadata.Force {
			log.Warnf("%s - continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	return nil
}

func (p *ValidateFacts) populateSan() {
	mgrs := p.config.Spec.Managers()
	for _, h := range mgrs {
		f := fmt.Sprintf("--san=%s", h.Address)
		p.config.Spec.Ucp.InstallFlags.Add(f)
		log.Warnf("%s: added manager node's public address to ucp installFlag SANs: %s", h, f)
	}
}

// validateDTRVersionJump validates UCP upgrade path
func (p *ValidateFacts) validateUCPVersionJump(conf *api.ClusterConfig) error {
	if conf.Spec.Ucp.Metadata.Installed && conf.Spec.Ucp.Metadata.InstalledVersion != "" {
		installedUCP, err := version.NewVersion(conf.Spec.Ucp.Metadata.InstalledVersion)
		if err != nil {
			return err
		}
		targetUCP, err := version.NewVersion(conf.Spec.Ucp.Version)
		if err != nil {
			return err
		}

		if installedUCP.GreaterThan(targetUCP) {
			return fmt.Errorf("can't downgrade UCP %s to %s", installedUCP.String(), targetUCP.String())
		}

		installedSegments := installedUCP.Segments()
		targetSegments := targetUCP.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("can't upgrade UCP directly from %s to %s - need to upgrade to %d.%d first", installedUCP.String(), targetUCP.String(), installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}

// validateDTRVersionJump validates DTR upgrade path
func (p *ValidateFacts) validateDTRVersionJump(conf *api.ClusterConfig) error {
	if conf.Spec.Dtr != nil && conf.Spec.Dtr.Metadata != nil && conf.Spec.Dtr.Metadata.Installed && conf.Spec.Dtr.Metadata.InstalledVersion != "" {
		installedDTR, err := version.NewVersion(conf.Spec.Dtr.Metadata.InstalledVersion)
		if err != nil {
			return err
		}
		targetDTR, err := version.NewVersion(conf.Spec.Dtr.Version)
		if err != nil {
			return err
		}

		if installedDTR.GreaterThan(targetDTR) {
			return fmt.Errorf("can't downgrade DTR %s to %s", installedDTR.String(), targetDTR.String())
		}

		installedSegments := installedDTR.Segments()
		targetSegments := targetDTR.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("can't upgrade DTR directly from %s to %s - need to upgrade to %d.%d first", installedDTR.String(), targetDTR.String(), installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}

// validateDataPlane checks if the calico data plane would get changed (VXLAN <-> VPIP)
func (p *ValidateFacts) validateDataPlane(conf *api.ClusterConfig) error {
	log.Debug("validating data plane settings")

	idx := conf.Spec.Ucp.InstallFlags.Index("--calico-vxlan")
	if idx < 0 {
		return nil
	}

	val := conf.Spec.Ucp.InstallFlags.GetValue("--calico-vxlan")
	var valB bool
	if val == "" {
		valB = true
	} else {
		v, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		valB = v
	}

	// User has explicitly defined --calico-vxlan=false but there is a windows host in the config
	if !valB {
		if conf.Spec.Hosts.Include(func(h *api.Host) bool { return h.IsWindows() }) {
			return fmt.Errorf("calico IPIP can't be used on Windows")
		}

		log.Debug("no windows hosts found")
	}

	if !conf.Spec.Ucp.Metadata.Installed {
		log.Debug("no existing UCP installation")
		return nil
	}

	// User has explicitly defined --calico-vxlan=false but there is already a calico with vxlan
	if conf.Spec.Ucp.Metadata.VXLAN {
		log.Debug("ucp has been installed with calico + vxlan")
		if !valB {
			return fmt.Errorf("calico configured with VXLAN, can't automatically change to IPIP")
		}
	} else {
		log.Debug("ucp has been installed with calico + vpip")
		// User has explicitly defined --calico-vxlan=true but there is already a calico with ipip
		if valB {
			return fmt.Errorf("calico configured with IPIP, can't automatically change to VXLAN")
		}
	}

	log.Debug("data plane settings check passed")

	return nil
}
