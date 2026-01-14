package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
)

// CleanUp phase is used by reset for performing post-uninstall cleanups.
type CleanUp struct {
	phase.BasicPhase
}

// Title for the phase.
func (p *CleanUp) Title() string {
	return "Clean up"
}

// Run does all the prep work on the hosts in parallel.
func (p *CleanUp) Run() error {
	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.cleanupEnv)
	if err != nil {
		return fmt.Errorf("failed to cleanup environment: %w", err)
	}

	return nil
}

func (p *CleanUp) cleanupEnv(h *mkeconfig.Host, _ *mkeconfig.ClusterConfig) error {
	if len(h.Environment) > 0 {
		if err := h.Configurer.CleanupEnvironment(h, h.Environment); err != nil {
			return fmt.Errorf("failed to cleanup environment: %w", err)
		}
	}
	return nil
}
