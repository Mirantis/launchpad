package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/msr"
	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
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
	if err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.updateEnvironment); err != nil {
		return fmt.Errorf("failed to update environment variables: %w", err)
	}

	if err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.prepareHost); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}

	if err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.authorizeDocker); err != nil {
		return fmt.Errorf("failed to authorize docker: %w", err)
	}

	if p.Config.Spec.ContainsMSR() && p.Config.Spec.MSR.ReplicaIDs == "sequential" {
		err := msr.AssignSequentialReplicaIDs(p.Config)
		if err != nil {
			return fmt.Errorf("failed to assign sequential MSR replica IDs: %w", err)
		}
	}

	// @TODO detect docker-ee version

	return nil
}

// Run the configurer specific preparation.
func (p *PrepareHost) prepareHost(h *mkeconfig.Host, _ *mkeconfig.ClusterConfig) error {
	if err := h.Configurer.PrepareHost(h); err != nil {
		log.Errorf("%s: %s", h, err)
		return fmt.Errorf("prepare host: %w", err)
	}
	return nil
}

func (p *PrepareHost) updateEnvironment(h *mkeconfig.Host, _ *mkeconfig.ClusterConfig) error {
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

func (p *PrepareHost) authorizeDocker(h *mkeconfig.Host, _ *mkeconfig.ClusterConfig) error {
	if err := h.AuthorizeDocker(); err != nil {
		return fmt.Errorf("failed to authorize docker: %w", err)
	}
	return nil
}
