package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// PrepareHost phase implementation does all the prep work we need for the hosts.
type PrepareHost struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *PrepareHost) Title() string {
	return "Prepare hosts"
}

// Run does all the prep work on the hosts in parallel.
func (p *PrepareHost) Run() error {
	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.updateEnvironment)
	if err != nil {
		return fmt.Errorf("failed to update environment variables: %w", err)
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.installBasePackages)
	if err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.fixContainerized)
	if err != nil {
		return fmt.Errorf("failed to apply containerized host fix: %w", err)
	}

	err = phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.authorizeDocker)
	if err != nil {
		return fmt.Errorf("failed to authorize docker: %w", err)
	}

	if p.Config.Spec.ContainsMSR() && p.Config.Spec.MSR.ReplicaIDs == "sequential" {
		err = msr.AssignSequentialReplicaIDs(p.Config)
		if err != nil {
			return fmt.Errorf("failed to assign sequential MSR replica IDs: %w", err)
		}
	}

	return nil
}

func (p *PrepareHost) installBasePackages(h *api.Host, _ *api.ClusterConfig) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing base packages", h)
			if err := h.Configurer.InstallMKEBasePackages(h); err != nil {
				log.Errorf("%s: %s", h, err)
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(5),
	)
	if err != nil {
		return fmt.Errorf("retry count exceeded: %w", err)
	}

	log.Infof("%s: base packages installed", h)
	return nil
}

func (p *PrepareHost) updateEnvironment(h *api.Host, _ *api.ClusterConfig) error {
	if len(h.Environment) > 0 {
		log.Infof("%s: updating environment", h)
		if err := h.Configurer.UpdateEnvironment(h, h.Environment); err != nil {
			return fmt.Errorf("failed to update environment variables: %w", err)
		}
		return nil
	}

	log.Debugf("%s: no environment variables specified for the host", h)
	return nil
}

func (p *PrepareHost) fixContainerized(h *api.Host, _ *api.ClusterConfig) error {
	if h.Configurer.IsContainer(h) {
		log.Infof("%s: is a container, applying a fix", h)
		if err := h.Configurer.FixContainer(h); err != nil {
			return fmt.Errorf("failed to apply containerized host fix: %w", err)
		}
	}
	return nil
}

func (p *PrepareHost) authorizeDocker(h *api.Host, config *api.ClusterConfig) error {
	if err := h.Configurer.AuthorizeDocker(h, config.Spec.MCR); err != nil {
		return fmt.Errorf("failed to authorize docker: %w", err)
	}
	return nil
}
