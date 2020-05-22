package phase

import (
	"github.com/Mirantis/mcc/pkg/config"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct {
	Analytics
}

// Title for the phase
func (p *Connect) Title() string {
	return "Open SSH Connection"
}

// Run connects to all the hosts in parallel
func (p *Connect) Run(config *config.ClusterConfig) error {
	return runParallelOnHosts(config.Hosts, config, p.connectHost)
}

func (p *Connect) connectHost(host *config.Host, c *config.ClusterConfig) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: opening SSH connection", host.Address)
			err := host.Connect()
			if err != nil {
				log.Errorf("%s: failed to connect -> %s", host.Address, err.Error())
			}
			return err
		},
	)
	if err != nil {
		log.Errorf("%s: failed to open connection", host.Address)
		return err
	}

	log.Printf("%s: SSH connection opened", host.Address)
	return nil
}
