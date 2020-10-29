package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/swarm"

	log "github.com/sirupsen/logrus"
)

// InitSwarm phase implementation
type InitSwarm struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *InitSwarm) Title() string {
	return "Initialize Swarm"
}

// Run initializes the swarm on the leader or skips if swarm is already initialized
func (p *InitSwarm) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()

	if !swarm.IsSwarmNode(swarmLeader) {
		log.Infof("%s: initializing swarm", swarmLeader.Address)
		output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm init --advertise-addr=%s", swarmLeader.SwarmAddress()))
		if err != nil {
			return fmt.Errorf("failed to initialize swarm: %s", output)
		}
		log.Infof("%s: swarm initialized successfully", swarmLeader.Address)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader.Address)
	}

	mgrToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token manager -q"))
	if err != nil {
		return fmt.Errorf("%s: failed to get swarm manager join-token: %s", swarmLeader.Address, err.Error())
	}
	p.config.Spec.Ucp.Metadata.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token worker -q"))
	if err != nil {
		return fmt.Errorf("%s: failed to get swarm worker join-token: %s", swarmLeader.Address, err.Error())
	}
	p.config.Spec.Ucp.Metadata.WorkerJoinToken = workerToken
	return nil
}
