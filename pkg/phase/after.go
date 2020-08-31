package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
)

// After phase implementation does all the prep work we need for the hosts
type After struct {
	Analytics
}

// Title for the phase
func (p *After) Title() string {
	return "Run After Hooks"
}

// Run does all the prep work on the hosts in parallel
func (p *After) Run(config *api.ClusterConfig) error {
	hosts := config.Spec.Hosts.Filter(func(h *api.Host) bool {
		return len(h.After) > 0
	})
	return hosts.ParallelEach(func(h *api.Host) error {
		return h.ExecAll(h.After)
	})
}
