package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// ConfigureEngine phase implementation
type ConfigureEngine struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that need their engine to be restarted
func (p *ConfigureEngine) HostFilterFunc(h *api.Host) bool {
	return len(h.DaemonConfig) > 0
}

// Prepare collects the hosts
func (p *ConfigureEngine) Prepare(config *api.ClusterConfig) error {
	p.Config = config
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *ConfigureEngine) Title() string {
	return "Configure Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *ConfigureEngine) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.Engine.Version,
	}
	return p.Hosts.ParallelEach(func(h *api.Host) error {
		log.Infof("%s: configuring engine", h)
		return h.ConfigureEngine()
	})
}
