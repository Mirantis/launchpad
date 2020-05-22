package phase

import (
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// Disconnect phase implementation
type Disconnect struct {
	Analytics
}

// Title for the phase
func (p *Disconnect) Title() string {
	return "Close SSH Connection"
}

// Run disconnects from all the hosts
func (p *Disconnect) Run(config *config.ClusterConfig) error {
	return runParallelOnHosts(config.Hosts, config, p.disconnectHost)
}

func (p *Disconnect) disconnectHost(host *config.Host, c *config.ClusterConfig) error {
	host.Connect()
	log.Printf("%s: SSH connection closed", host.Address)
	return nil
}
