package phase

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

// ValidateFacts phase implementation to validate facts from config and collected metadata
type ValidateFacts struct {
	Analytics
}

// Title for the phase
func (p *ValidateFacts) Title() string {
	return "Validate Facts"
}

// Run collect all the facts from hosts in parallel
func (p *ValidateFacts) Run(conf *api.ClusterConfig) error {
	if err := p.validateUCPVersionJump(conf); err != nil {
		if conf.Spec.Metadata.Force {
			log.Warnf("%s - continuing anyway because --force given", err.Error())
		} else {
			return err
		}
	}

	if err := p.validateDTRVersionJump(conf); err != nil {
		if conf.Spec.Metadata.Force {
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
			return fmt.Errorf("Can't downgrade UCP %s to %s", installedUCP.String(), targetUCP.String())
		}

		installedSegments := installedUCP.Segments()
		targetSegments := targetUCP.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("Can't upgrade UCP directly from %s to %s - need to upgrade to %d.%d first.", installedUCP.String(), targetUCP.String(), installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}

// validateDTRVersionJump validates DTR upgrade path
func (p *ValidateFacts) validateDTRVersionJump(conf *api.ClusterConfig) error {
	if conf.Spec.Dtr.Metadata.Installed && conf.Spec.Dtr.Metadata.InstalledVersion != "" {
		installedDTR, err := version.NewVersion(conf.Spec.Dtr.Metadata.InstalledVersion)
		if err != nil {
			return err
		}
		targetDTR, err := version.NewVersion(conf.Spec.Dtr.Version)
		if err != nil {
			return err
		}

		if installedDTR.GreaterThan(targetDTR) {
			return fmt.Errorf("Can't downgrade DTR %s to %s", installedDTR.String(), targetDTR.String())
		}

		installedSegments := installedDTR.Segments()
		targetSegments := targetDTR.Segments()

		// This will fail if there's something like 2.x => 3.x or 3.x => 4.x.
		if installedSegments[0] == targetSegments[0] && targetSegments[1]-installedSegments[1] > 1 {
			return fmt.Errorf("Can't upgrade DTR directly from %s to %s - need to upgrade to %d.%d first.", installedDTR.String(), targetDTR.String(), installedSegments[0], installedSegments[1]+1)
		}
	}

	return nil
}
