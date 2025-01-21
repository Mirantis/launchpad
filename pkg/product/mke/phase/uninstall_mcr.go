package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/mcr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// UninstallMCR phase implementation.
type UninstallMCR struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *UninstallMCR) Title() string {
	return "Uninstall Mirantis Container Runtime from the hosts"
}

// Run installs the engine on each host.
func (p *UninstallMCR) Run() error {
	if err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.uninstallMCR); err != nil {
		return fmt.Errorf("uninstall container runtime: %w", err)
	}
	return nil
}

func (p *UninstallMCR) uninstallMCR(h *api.Host, config *api.ClusterConfig) error {
	log.Infof("%s: uninstalling container runtime", h)

	leader := config.Spec.SwarmLeader()

	if err := mcr.DrainNode(leader, h); err != nil {
		return fmt.Errorf("%s: drain node: %w", h, err)
	}

	uVolumeCmd := h.Configurer.DockerCommandf("volume prune -f")
	log.Infof("%s: unmounted dangling volumes", h)

	if err := h.Exec(uVolumeCmd); err != nil {
		return fmt.Errorf("%s: failed to unmount dangling volumes: %w", h, err)
	}

	if err := h.Configurer.UninstallMCR(h, h.Metadata.MCRInstallScript, config.Spec.MCR); err != nil {
		return fmt.Errorf("%s: uninstall container runtime: %w", h, err)
	}

	log.Infof("%s: mirantis container runtime uninstalled", h)

	return nil
}
