package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// RunHooks phase runs a set of hooks configured for the host
type RunHooks struct {
	phase.Analytics
	phase.HostSelectPhase
	StepListFunc func(*api.Host) *[]string
	Stage        string
	Action       string
}

// HostFilterFunc returns true for hosts that have non-empty list of hooks returned by the StepListFunc
func (p *RunHooks) HostFilterFunc(h *api.Host) bool {
	steps := p.StepListFunc(h)
	return len(*steps) > 0
}

// Prepare collects the hosts
func (p *RunHooks) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *RunHooks) Title() string {
	return fmt.Sprintf("Run %s %s Hooks", p.Stage, p.Action)
}

// Run does all the prep work on the hosts in parallel
func (p *RunHooks) Run() error {
	return p.Hosts.ParallelEach(func(h *api.Host) error {
		if steps := p.StepListFunc(h); steps != nil {
			return h.ExecAll(*steps)
		}
		return nil
	})
}
