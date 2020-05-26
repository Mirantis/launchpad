package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
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
func (p *Connect) Run(config *api.ClusterConfig) error {
	return runParallelOnHosts(config.Spec.Hosts, config, p.connectHost)
}

func (p *Connect) connectHost(host *api.Host, c *api.ClusterConfig) error {
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
