package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// UninstallEngine phase implementation
type UninstallEngine struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *UninstallEngine) Title() string {
	return "Uninstall Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *UninstallEngine) Run() error {
	return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.uninstallEngine)
}

func (p *UninstallEngine) uninstallEngine(h *api.Host, c *api.ClusterConfig) error {
	err := h.Exec(h.Configurer.DockerCommandf("info"))
	if err != nil {
		log.Infof("%s: engine not installed, skipping", h)
		return nil
	}
	log.Infof("%s: uninstalling engine", h)
	err = h.Configurer.UninstallEngine(&c.Spec.Engine)
	if err == nil {
		log.Infof("%s: engine uninstalled", h)
	}

	return err
}
