package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// Disconnect phase implementation
type Disconnect struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *Disconnect) Title() string {
	return "Close Connection"
}

// Run disconnects from all the hosts
func (p *Disconnect) Run() error {
	return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.disconnectHost)
}

func (p *Disconnect) disconnectHost(h *api.Host, c *api.ClusterConfig) error {
	h.Disconnect()
	log.Infof("%s: connection closed", h)
	return nil
}
