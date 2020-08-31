package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
)

// Before phase implementation does all the prep work we need for the hosts
type Before struct {
	Analytics
}

// Title for the phase
func (p *Before) Title() string {
	return "Run Before Hooks"
}

// Run does all the prep work on the hosts in parallel
func (p *Before) Run(config *api.ClusterConfig) error {
	hosts := config.Spec.Hosts.Filter(func(h *api.Host) bool {
		return len(h.Before) > 0
	})
	return hosts.ParallelEach(func(h *api.Host) error {
		return h.ExecAll(h.Before)
	})
}
