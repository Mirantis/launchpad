package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// PrepareHost phase implementation does all the prep work we need for the hosts
type PrepareHost struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *PrepareHost) Title() string {
	return "Prepare hosts"
}

// Run does all the prep work on the hosts in parallel
func (p *PrepareHost) Run() error {
	err := runParallelOnHosts(p.config.Spec.Hosts, p.config, p.installBasePackages)
	if err != nil {
		return err
	}

	err = runParallelOnHosts(p.config.Spec.Hosts, p.config, p.updateEnvironment)
	if err != nil {
		return err
	}

	return nil
}

func (p *PrepareHost) installBasePackages(host *api.Host, c *api.ClusterConfig) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing base packages", host.Address)
			err := host.Configurer.InstallBasePackages()

			return err
		},
	)
	if err != nil {
		log.Errorf("%s: failed to install base packages -> %s", host.Address, err.Error())
		return err
	}

	log.Printf("%s: base packages installed", host.Address)
	return nil
}

func (p *PrepareHost) updateEnvironment(host *api.Host, c *api.ClusterConfig) error {
	if len(host.Environment) > 0 {
		log.Infof("%s: updating environment", host.Address)
		return host.Configurer.UpdateEnvironment()
	}

	log.Debugf("%s: no environment variables specified for the host", host.Address)
	return nil
}
