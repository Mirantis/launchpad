package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
)

// CleanUp phase implementation does all the prep work we need for the hosts
type CleanUp struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *CleanUp) Title() string {
	return "Clean up"
}

// Run does all the prep work on the hosts in parallel
func (p *CleanUp) Run() error {
	err := runParallelOnHosts(p.config.Spec.Hosts, p.config, p.cleanupEnv)
	if err != nil {
		return err
	}

	return nil
}

func (p *CleanUp) cleanupEnv(h *api.Host, c *api.ClusterConfig) error {
	if len(h.Environment) > 0 {
		return h.Configurer.CleanupEnvironment()
	}
	return nil
}
