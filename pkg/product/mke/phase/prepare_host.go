package phase

import (
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// PrepareHost phase implementation does all the prep work we need for the hosts
type PrepareHost struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *PrepareHost) Title() string {
	return "Prepare hosts"
}

// Run does all the prep work on the hosts in parallel
func (p *PrepareHost) Run() error {
	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.updateEnvironment)
	if err != nil {
		return err
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.installBasePackages)
	if err != nil {
		return err
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.fixContainerized)
	if err != nil {
		return err
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.authorizeDocker)
	if err != nil {
		return err
	}

	if p.Config.Spec.ContainsMSR() && p.Config.Spec.MSR.ReplicaIDs == "sequential" {
		err = msr.AssignSequentialReplicaIDs(p.Config)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PrepareHost) installBasePackages(h *api.Host, c *api.ClusterConfig) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing base packages", h)
			err := h.Configurer.InstallMKEBasePackages(h)

			return err
		},
	)
	if err != nil {
		log.Errorf("%s: failed to install base packages -> %s", h, err.Error())
		return err
	}

	log.Infof("%s: base packages installed", h)
	return nil
}

func (p *PrepareHost) updateEnvironment(h *api.Host, c *api.ClusterConfig) error {
	if len(h.Environment) > 0 {
		log.Infof("%s: updating environment", h)
		return h.Configurer.UpdateEnvironment(h, h.Environment)
	}

	log.Debugf("%s: no environment variables specified for the host", h)
	return nil
}

func (p *PrepareHost) fixContainerized(h *api.Host, c *api.ClusterConfig) error {
	if h.Configurer.IsContainer(h) {
		log.Infof("%s: is a container, applying a fix", h)
		return h.Configurer.FixContainer(h)
	}
	return nil
}

func (p *PrepareHost) authorizeDocker(h *api.Host, c *api.ClusterConfig) error {
	return h.Configurer.AuthorizeDocker(h)
}
