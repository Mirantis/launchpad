package phase

import (
	"fmt"
	"regexp"
	"strconv"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"

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

// Run collect all the facts from hosts in parallel
func (p *ValidateFacts) Run() error {
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
	if conf.Spec.Dtr.Metadata != nil && conf.Spec.Dtr.Metadata.Installed && conf.Spec.Dtr.Metadata.InstalledVersion != "" {
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

	re := regexp.MustCompile(`^--calico-vxlan=?(.*)`)

	hasTrue := false
	hasFalse := false

	for _, v := range conf.Spec.Ucp.InstallFlags {
		match := re.FindStringSubmatch(v)
		if len(match) == 2 {
			if match[1] == "" {
				hasTrue = true
			} else {
				b, err := strconv.ParseBool(match[1])
				if err != nil {
					return fmt.Errorf("invalid --calico-vxlan value %s", v)
				}
				if b {
					hasTrue = true
				} else {
					hasFalse = true
				}
			}
		}
	}

	// User has explicitly defined --calico-vxlan=false but there is a windows host in the config
	if hasFalse {
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
		if hasFalse {
			return fmt.Errorf("calico configured with VXLAN, can't automatically change to IPIP")
		}
	} else {
		log.Debug("ucp has been installed with calico + vpip")
		// User has explicitly defined --calico-vxlan=true but there is already a calico with ipip
		if hasTrue {
			return fmt.Errorf("calico configured with IPIP, can't automatically change to VXLAN")
		}
	}

	log.Debug("data plane settings check passed")

	return nil
}
