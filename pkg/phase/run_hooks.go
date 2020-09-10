package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	log "github.com/sirupsen/logrus"
)

// RunHooks phase runs a set of hooks configured for the host
type RunHooks struct {
	Analytics
	HostSelectPhase
	StepListFunc func(*api.Host) *[]string
	Stage        string
	Action       string
}

// HostFilterFunc returns true for hosts that have non-empty list of hooks returned by the StepListFunc
func (p *RunHooks) HostFilterFunc(host *api.Host) bool {
	steps := p.StepListFunc(host)
	return len(*steps) > 0
}

// Prepare collects the hosts
func (p *RunHooks) Prepare(config *api.ClusterConfig) error {
	p.config = config
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.hosts = hosts
	return nil
}

// Title for the phase
func (p *RunHooks) Title() string {
	return fmt.Sprintf("Run %s %s Hooks", p.Stage, p.Action)
}

// Run does all the prep work on the hosts in parallel
func (p *RunHooks) Run() error {
	return p.hosts.ParallelEach(func(h *api.Host) error {
		if steps := p.StepListFunc(h); steps != nil {
			return h.ExecAll(*steps)
		}
		return nil
	})
}
