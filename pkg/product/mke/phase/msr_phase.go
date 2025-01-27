package phase

import (
	github.com/Mirantis/launchpad/pkg/phase"
)

// MSRPhase only runs when the config includes MSR hosts.
type MSRPhase struct {
	phase.BasicPhase
}

// ShouldRun default implementation for MSR phase returns true when the config has MSR nodes.
func (p *MSRPhase) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR()
}
