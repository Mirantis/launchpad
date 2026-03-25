package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	log "github.com/sirupsen/logrus"
)

// InstallMCR phase implementation.
type InstallMCR struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that do not have engine installed.
func (p *InstallMCR) HostFilterFunc(h *mkeconfig.Host) bool {
	return h.Metadata.MCRVersion == ""
}

// Prepare collects the hosts.
func (p *InstallMCR) Prepare(config any) error {
	cfg, ok := config.(*mkeconfig.ClusterConfig)
	if !ok {
		return errInvalidConfig
	}

	p.Config = cfg
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase.
func (p *InstallMCR) Title() string {
	return "Install Mirantis Container Runtime on the hosts"
}

// Run installs the engine on each host.
func (p *InstallMCR) Run() error {
	p.EventProperties = map[string]any{
		"engine_channel": p.Config.Spec.MCR.Channel,
	}

	if err := p.Hosts.ParallelEach(p.installMCR); err != nil {
		return fmt.Errorf("failed to install container runtime: %w", err)
	}
	return nil
}

func (p *InstallMCR) installMCR(h *mkeconfig.Host) error {
	log.Infof("%s: installing container runtime (%s)", h, p.Config.Spec.MCR.Channel)
	if err := h.Configurer.InstallMCR(h, p.Config.Spec.MCR); err != nil {
		log.Errorf("%s: failed to install container runtime: %s", h, err.Error())
		return fmt.Errorf("%s: failed to install container runtime: %w", h, err)
	}

	if err := h.AuthorizeDocker(); err != nil {
		return fmt.Errorf("%s: failed to authorize docker: %w", h, err)
	}

	log.Infof("%s: mcr installed", h)
	h.Metadata.MCRInstalled = true
	h.Metadata.MCRRestartRequired = false // we just installed, so a restart is not required
	return nil
}
