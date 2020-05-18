package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/config"

	log "github.com/sirupsen/logrus"
)

// InitSwarm phase implementation
type InitSwarm struct{}

// Title for the phase
func (p *InitSwarm) Title() string {
	return "Initialize Swarm"
}

// Run initializes the swarm on the leader or skips if swarm is already initialized
func (p *InitSwarm) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Managers()[0]

	if !util.IsSwarmNode(swarmLeader) {
		log.Infof("%s: initializing swarm", swarmLeader.Address)
		err := swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("swarm init --advertise-addr=%s", swarmLeader.SwarmAddress()))
		if err != nil {
			return NewError(fmt.Sprintf("Failed to initialize swarm: %s", err.Error()))
		}
		log.Infof("%s: swarm initialized succesfully", swarmLeader.Address)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader.Address)
	}

	mgrToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token manager -q"))
	if err != nil {
		return NewError("failed to get swarm manager join-token")
	}
	config.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token worker -q"))
	if err != nil {
		return NewError("failed to get swarm worker join-token")
	}
	config.WorkerJoinToken = workerToken

	return nil
}
