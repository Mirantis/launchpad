package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct {
	Analytics
}

// Title for the phase
func (p *Connect) Title() string {
	return "Open Remote Connection"
}

// Run connects to all the hosts in parallel
func (p *Connect) Run(config *api.ClusterConfig) error {
	return runParallelOnHosts(config.Spec.Hosts, config, p.connectHost)
}

func (p *Connect) connectHost(host *api.Host, c *api.ClusterConfig) error {
	proto := "SSH"

	if host.WinRM != nil {
		proto = "WinRM"
	}

	err := retry.Do(
		func() error {
			log.Infof("%s: opening %s connection", host.Address, proto)
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

	log.Printf("%s: %s connection opened", host.Address, proto)
	return nil
}
