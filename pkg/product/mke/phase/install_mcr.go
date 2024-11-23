package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/mcr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// InstallMCR phase implementation.
type InstallMCR struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that do not have engine installed.
func (p *InstallMCR) HostFilterFunc(h *api.Host) bool {
	return h.Metadata.MCRVersion == ""
}

// Prepare collects the hosts.
func (p *InstallMCR) Prepare(config interface{}) error {
	cfg, ok := config.(*api.ClusterConfig)
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
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.MCR.Version,
	}

	if err := p.Hosts.ParallelEach(p.installMCR); err != nil {
		return fmt.Errorf("failed to install container runtime: %w", err)
	}
	return nil
}

func (p *InstallMCR) installMCR(h *api.Host) error {
	if err := retry.Do(
		func() error {
			log.Infof("%s: installing container runtime (%s)", h, p.Config.Spec.MCR.Version)
			if err := h.Configurer.InstallMCR(h, h.Metadata.MCRInstallScript, p.Config.Spec.MCR); err != nil {
				log.Errorf("%s: failed to install container runtime: %s", h, err.Error())
				return fmt.Errorf("%s: failed to install container runtime: %w", h, err)
			}
			return nil
		},
	); err != nil {
		return fmt.Errorf("retry count exceeded: %w", err)
	}

	if err := h.AuthorizeDocker(); err != nil {
		return fmt.Errorf("%s: failed to authorize docker: %w", h, err)
	}

	// check MCR is running, maybe rebooting and updating metadata (don't bother restarting MCR, as we just installed/started it
	if err := mcr.EnsureMCRVersion(h, p.Config.Spec.MCR.Version, false); err != nil {
		return fmt.Errorf("failed while attempting to ensure the installed version %w", err)
	}

	h.Metadata.MCRInstalled = true
	return nil
}
