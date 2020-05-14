package phase

import (
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

type Disconnect struct{}

func (p *Disconnect) Title() string {
	return "Close SSH Connection"
}

func (p *Disconnect) Run(config *config.ClusterConfig) *PhaseError {
	return runParallelOnHosts(config.Hosts, config, p.disconnectHost)
}

func (p *Disconnect) disconnectHost(host *config.Host, c *config.ClusterConfig) error {
	host.Connect()
	log.Printf("%s: SSH connection closed", host.Address)
	return nil
}
