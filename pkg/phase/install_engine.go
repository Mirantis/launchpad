package phase

import (
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	retry "github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

type InstallEngine struct{}

func (p *InstallEngine) Title() string {
	return "Install Docker EE Engine on the hosts"
}

func (p *InstallEngine) Run(config *config.ClusterConfig) error {
	var wg sync.WaitGroup
	for _, host := range config.Hosts {
		wg.Add(1)
		go p.installEngine(host, &config.Engine, &wg)
	}
	wg.Wait()

	return nil
}

func (p *InstallEngine) installEngine(host *config.Host, engineConfig *config.EngineConfig, wg *sync.WaitGroup) error {
	defer wg.Done()
	err := retry.Do(
		func() error {
			logrus.Infof("%s: installing base packages", host.Address)
			err := host.Configurer.InstallBasePackages()
			logrus.Infof("%s: installing engine", host.Address)
			err = host.Configurer.InstallEngine(engineConfig)

			return err
		},
	)
	if err != nil {
		logrus.Errorf("%s: failed to install engine -> %s", host.Address, err.Error())
		return err
	}

	logrus.Printf("%s: Engine installed", host.Address)
	return nil
}
