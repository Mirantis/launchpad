package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	log "github.com/sirupsen/logrus"
)

// ConfigureK0s phase
type ConfigureK0s struct {
	phase.Analytics
	BasicPhase
}

// Title for the phase
func (p *ConfigureK0s) Title() string {
	return "Configure K0s on hosts"
}

// Run ...
func (p *ConfigureK0s) Run() error {
	return RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.writeConfig)
}

func (p *ConfigureK0s) writeConfig(h *api.Host, c *api.ClusterConfig) error {
	if h.Role == "server" {
		log.Infof("%s: writing K0s config", h)

		return h.ConfigureK0s(&p.Config.Spec.K0s.Config)
	}
	return nil
}
