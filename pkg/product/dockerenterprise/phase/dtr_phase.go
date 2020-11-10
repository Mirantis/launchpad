package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
)

// DtrPhase only runs when the config includes dtr hosts
type DtrPhase struct {
	phase.BasicPhase
}

// ShouldRun default implementation for DTR phase returns true when the config has DTR nodes
func (p *DtrPhase) ShouldRun() bool {
	return p.Config.Spec.ContainsDtr()
}
