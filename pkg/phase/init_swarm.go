package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/config"

	log "github.com/sirupsen/logrus"
)

type InitSwarm struct{}

func (p *InitSwarm) Title() string {
	return "Initialize Swarm"
}

func (p *InitSwarm) Run(config *config.ClusterConfig) *PhaseError {
	swarmLeader := config.Controllers()[0]

	if !util.IsSwarmNode(swarmLeader) {
		log.Debugf("%s: initializing swarm", swarmLeader.Address)
		err := swarmLeader.Exec(fmt.Sprintf("sudo docker swarm init --advertise-addr=%s", swarmLeader.SwarmAddress()))
		if err != nil {
			return NewPhaseError(fmt.Sprintf("Failed to initialize swarm: %s", err.Error()))
		}
		log.Debugf("%s: swarm initilized succesfully", swarmLeader.Address)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader.Address)
	}

	mgrToken, err := swarmLeader.ExecWithOutput("sudo docker swarm join-token manager -q")
	if err != nil {
		return NewPhaseError("failed to get swarm manager join-token")
	}
	config.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecWithOutput("sudo docker swarm join-token worker -q")
	if err != nil {
		return NewPhaseError("failed to get swarm manager join-token")
	}
	config.WorkerJoinToken = workerToken

	return nil
}
