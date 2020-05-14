package phase

import (
	"github.com/Mirantis/mcc/pkg/config"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

type InstallEngine struct{}

func (p *InstallEngine) Title() string {
	return "Install Docker EE Engine on the hosts"
}

func (p *InstallEngine) Run(config *config.ClusterConfig) *PhaseError {
	return runParallelOnHosts(config.Hosts, config, p.installEngine)
}

func (p *InstallEngine) installEngine(host *config.Host, c *config.ClusterConfig) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing engine", host.Address)
			err := host.Configurer.InstallEngine(&c.Engine)

			return err
		},
	)
	if err != nil {
		log.Errorf("%s: failed to install engine -> %s", host.Address, err.Error())
		return err
	}

	log.Printf("%s: Engine installed", host.Address)
	return nil
}
