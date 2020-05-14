package phase

import (
	"github.com/Mirantis/mcc/pkg/config"
	retry "github.com/avast/retry-go"
	"github.com/sirupsen/logrus"
)

type PrepareHost struct{}

func (p *PrepareHost) Title() string {
	return "Prepare hosts"
}

func (p *PrepareHost) Run(config *config.ClusterConfig) *PhaseError {
	return runParallelOnHosts(config.Hosts, config, p.prepareHost)
}

func (p *PrepareHost) prepareHost(host *config.Host, c *config.ClusterConfig) error {
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
