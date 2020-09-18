package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
	log "github.com/sirupsen/logrus"
)

// Disconnect phase implementation
type Disconnect struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *Disconnect) Title() string {
	return "Close Connection"
}

// Run disconnects from all the hosts
func (p *Disconnect) Run() error {
	return runParallelOnHosts(p.config.Spec.Hosts, p.config, p.disconnectHost)
}

func (p *Disconnect) disconnectHost(host *api.Host, c *api.ClusterConfig) error {
	host.Disconnect()
	log.Printf("%s: connection closed", host.Address)
	return nil
}
