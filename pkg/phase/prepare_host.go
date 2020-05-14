package phase

import (
	"sync"

	"github.com/Mirantis/mcc/pkg/config"
	retry "github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

type PrepareHost struct{}

func (p *PrepareHost) Title() string {
	return "Prepare hosts"
}

func (p *PrepareHost) Run(config *config.ClusterConfig) error {
	var wg sync.WaitGroup
	for _, host := range config.Hosts {
		wg.Add(1)
		go p.prepareHost(host, &wg)
	}
	wg.Wait()

	return nil
}

func (p *PrepareHost) prepareHost(host *config.Host, wg *sync.WaitGroup) error {
	defer wg.Done()
	err := retry.Do(
		func() error {
			logrus.Infof("%s: installing base packages", host.Address)
			err := host.Configurer.InstallBasePackages()

			return err
		},
	)
	if err != nil {
		logrus.Errorf("%s: failed to install base packages -> %s", host.Address, err.Error())
		return err
	}

	logrus.Printf("%s: base packages installed", host.Address)
	return nil
}
