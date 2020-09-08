package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
)

// CleanUp phase implementation does all the prep work we need for the hosts
type CleanUp struct {
	Analytics
}

// Title for the phase
func (p *CleanUp) Title() string {
	return "Clean up"
}

// Run does all the prep work on the hosts in parallel
func (p *CleanUp) Run(config *api.ClusterConfig) error {
	err := runParallelOnHosts(config.Spec.Hosts, config, p.cleanupEnv)
	if err != nil {
		return err
	}

	return nil
}

func (p *CleanUp) cleanupEnv(host *api.Host, c *api.ClusterConfig) error {
	if len(host.Environment) > 0 {
		return host.Configurer.CleanupEnvironment()
	}
	return nil
}
