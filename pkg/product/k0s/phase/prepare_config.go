package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	log "github.com/sirupsen/logrus"
)

//PrepareConfig phase
type PrepareConfig struct {
	phase.Analytics
	BasicPhase
}

// Title for the phase
func (p *PrepareConfig) Title() string {
	return "Prepare K0s config"
}

//Run ...
func (p *PrepareConfig) Run() error {
	return RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.prepareConfig)
}

func (p *PrepareConfig) prepareConfig(h *api.Host, c *api.ClusterConfig) error {
	if h.Role == "server" {
		log.Infof("%s: writing K0s config", h)

		return h.PrepareConfig(&p.Config.Spec.K0s.Config)
	}
	return nil
}
