package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// UninstallMCR phase implementation
type UninstallMCR struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *UninstallMCR) Title() string {
	return "Uninstall Mirantis Container Runtime from the hosts"
}

// Run installs the engine on each host
func (p *UninstallMCR) Run() error {
	return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.uninstallMCR)
}

func (p *UninstallMCR) uninstallMCR(h *api.Host, c *api.ClusterConfig) error {
	err := h.Exec(h.Configurer.DockerCommandf("info"))
	if err != nil {
		log.Infof("%s: container runtime not installed, skipping", h)
		return nil
	}
	log.Infof("%s: uninstalling container runtime", h)
	err = h.Configurer.UninstallMCR(h, h.Metadata.MCRInstallScript, c.Spec.MCR)
	if err == nil {
		log.Infof("%s: mirantis container runtime uninstalled", h)
	}

	return err
}
