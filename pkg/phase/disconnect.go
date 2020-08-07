package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	log "github.com/sirupsen/logrus"
)

// Disconnect phase implementation
type Disconnect struct {
	Analytics
}

// Title for the phase
func (p *Disconnect) Title() string {
	return "Close Connection"
}

// Run disconnects from all the hosts
func (p *Disconnect) Run(config *api.ClusterConfig) error {
	return runParallelOnHosts(config.Spec.Hosts, config, p.disconnectHost)
}

func (p *Disconnect) disconnectHost(host *api.Host, c *api.ClusterConfig) error {
	host.Disconnect()
	log.Printf("%s: connection closed", host.Address)
	return nil
}
