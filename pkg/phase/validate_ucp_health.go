package phase

// ValidateUcpHealth phase implementation
type ValidateUcpHealth struct {
	Analytics
	BasicPhase
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
	swarmLeader := p.config.Spec.SwarmLeader()
	return p.config.Spec.CheckUCPHealthLocal(swarmLeader)
}
