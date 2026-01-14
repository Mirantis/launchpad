package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	log "github.com/sirupsen/logrus"
)

// InstallMCRLicense phase implementation.
type InstallMCRLicense struct {
	phase.Analytics
	phase.HostSelectPhase
}

// ShouldRun if a license is provided, we should run this phase.
func (p *InstallMCRLicense) ShouldRun() bool {
	return p.Config.Spec.MCR.License != ""
}

// Title for the phase.
func (p *InstallMCRLicense) Title() string {
	return "Install Mirantis Container Runtime License"
}

// Run installs the engine on each host.
func (p *InstallMCRLicense) Run() error {
	if err := p.Hosts.ParallelEach(p.installMCRLicense); err != nil {
		return fmt.Errorf("failed to install MCR license: %w", err)
	}
	return nil
}

func (p *InstallMCRLicense) installMCRLicense(h *mkeconfig.Host) error {
	log.Infof("%s: installing MCR license", h)
	if err := h.Configurer.InstallMCRLicense(h, p.Config.Spec.MCR.License); err != nil {
		log.Errorf("%s: failed to install MCR License: %s", h, err.Error())
		return fmt.Errorf("%s: failed to install MCR License: %w", h, err)
	}

	return nil
}
