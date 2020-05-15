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
	swarmLeader := config.Controllers()[0]

	if !util.IsSwarmNode(swarmLeader) {
		log.Debugf("%s: initializing swarm", swarmLeader.Address)
		err := swarmLeader.Execf("sudo docker swarm init --advertise-addr=%s", swarmLeader.SwarmAddress())
		if err != nil {
			return NewError(fmt.Sprintf("Failed to initialize swarm: %s", err.Error()))
		}
		log.Debugf("%s: swarm initilized succesfully", swarmLeader.Address)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader.Address)
	}

	mgrToken, err := swarmLeader.ExecWithOutput("sudo docker swarm join-token manager -q")
	if err != nil {
		return NewError("failed to get swarm manager join-token")
	}
	config.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecWithOutput("sudo docker swarm join-token worker -q")
	if err != nil {
		return NewError("failed to get swarm manager join-token")
	}
	config.WorkerJoinToken = workerToken

	return nil
}
