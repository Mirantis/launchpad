package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	log "github.com/sirupsen/logrus"
)

// ConfigureMCR phase implementation.
type ConfigureMCR struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that need their engine to be restarted.
func (p *ConfigureMCR) HostFilterFunc(h *mkeconfig.Host) bool {
	return len(h.DaemonConfig) > 0
}

// Prepare collects the hosts.
func (p *ConfigureMCR) Prepare(config interface{}) error {
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
func (p *ConfigureMCR) Title() string {
	return "Configure Mirantis Container Runtime on the hosts"
}

// Run installs the engine on each host.
func (p *ConfigureMCR) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.MCR.Version,
	}
	err := p.Hosts.ParallelEach(func(h *mkeconfig.Host) error {
		log.Infof("%s: configuring container runtime", h)
		if err := h.ConfigureMCR(); err != nil {
			return fmt.Errorf("failed to configure container runtime on %s: %w", h, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to configure container runtime: %w", err)
	}
	return nil
}
