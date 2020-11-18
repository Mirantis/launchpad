package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
)

// ValidateMKEHealth validates MKE health locally from the MKE leader
type ValidateMKEHealth struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *ValidateMKEHealth) Title() string {
	return "Validating MKE Health"
}

// Run validates the health of MKE is sane before continuing with other
// launchpad phases, should be used when installing products that depend
// on MKE, such as MSR
func (p *ValidateMKEHealth) Run() error {
	// Issue a health check to the MKE san host until we receive an 'ok' status
	swarmLeader := p.Config.Spec.SwarmLeader()
	return p.Config.Spec.CheckMKEHealthLocal(swarmLeader)
}
