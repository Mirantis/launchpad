package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
)

// ValidateUcpHealth validates UCP health locally from the UCP leader
type ValidateUcpHealth struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *ValidateUcpHealth) Title() string {
	return "Validating UCP Health"
}

// Run validates the health of UCP is sane before continuing with other
// launchpad phases, should be used when installing products that depend
// on UCP, such as DTR
func (p *ValidateUcpHealth) Run() error {
	// Issue a health check to the UCP san host until we receive an 'ok' status
	swarmLeader := p.Config.Spec.SwarmLeader()
	return p.Config.Spec.CheckUCPHealthLocal(swarmLeader)
}
