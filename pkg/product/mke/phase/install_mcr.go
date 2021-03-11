package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// InstallMCR phase implementation
type InstallMCR struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that do not have engine installed
func (p *InstallMCR) HostFilterFunc(h *api.Host) bool {
	return h.Metadata.MCRVersion == ""
}

// Prepare collects the hosts
func (p *InstallMCR) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *InstallMCR) Title() string {
	return "Install Mirantis Container Runtime on the hosts"
}

// Run installs the engine on each host
func (p *InstallMCR) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.MCR.Version,
	}

	return p.Hosts.ParallelEach(p.installMCR)
}

func (p *InstallMCR) installMCR(h *api.Host) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing container runtime (%s)", h, p.Config.Spec.MCR.Version)
			return h.Configurer.InstallMCR(h, h.Metadata.MCRInstallScript, p.Config.Spec.MCR)
		},
	)
	if err != nil {
		log.Errorf("%s: failed to install container runtime -> %s", h, err.Error())
		return err
	}

	if err := h.Configurer.AuthorizeDocker(h); err != nil {
		return err
	}

	currentVersion, err := h.MCRVersion()
	if err != nil {
		if err := h.Reboot(); err != nil {
			return err
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after installation: %s", h, err.Error())
		}
	}

	if currentVersion != p.Config.Spec.MCR.Version {
		err = h.Configurer.RestartMCR(h)
		if err != nil {
			return fmt.Errorf("%s: failed to restart container runtime", h)
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after restart: %s", h, err.Error())
		}
	}

	if currentVersion != p.Config.Spec.MCR.Version {
		return fmt.Errorf("%s: container runtime version not %s after installation", h, p.Config.Spec.MCR.Version)
	}

	log.Infof("%s: mirantis container runtime version %s installed", h, p.Config.Spec.MCR.Version)
	h.Metadata.MCRVersion = p.Config.Spec.MCR.Version
	h.Metadata.MCRRestartRequired = false
	return nil
}
