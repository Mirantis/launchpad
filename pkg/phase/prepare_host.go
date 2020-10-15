package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
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
	err := runParallelOnHosts(p.config.Spec.Hosts, p.config, p.updateEnvironment)
	if err != nil {
		return err
	}

	err = runParallelOnHosts(p.config.Spec.Hosts, p.config, p.installBasePackages)
	if err != nil {
		return err
	}

	return nil
}

func (p *PrepareHost) installBasePackages(h *api.Host, c *api.ClusterConfig) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing base packages", h.Address)
			err := h.Configurer.InstallBasePackages()

			return err
		},
	)
	if err != nil {
		log.Errorf("%s: failed to install base packages -> %s", h.Address, err.Error())
		return err
	}

	log.Printf("%s: base packages installed", h.Address)
	return nil
}

func (p *PrepareHost) updateEnvironment(h *api.Host, c *api.ClusterConfig) error {
	if len(h.Environment) > 0 {
		log.Infof("%s: updating environment", h.Address)
		return h.Configurer.UpdateEnvironment()
	}

	log.Debugf("%s: no environment variables specified for the host", h.Address)
	return nil
}
