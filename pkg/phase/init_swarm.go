package phase

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/swarm"

	log "github.com/sirupsen/logrus"
)

// InitSwarm phase implementation
type InitSwarm struct {
	Analytics
}

// Title for the phase
func (p *InitSwarm) Title() string {
	return "Initialize Swarm"
}

// Run initializes the swarm on the leader or skips if swarm is already initialized
func (p *InitSwarm) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	if !swarm.IsSwarmNode(swarmLeader) {
		log.Infof("%s: initializing swarm", swarmLeader.Address)
		output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm init --advertise-addr=%s", swarmLeader.SwarmAddress()), exec.Redact("--token .*"))
		if err != nil {
			return NewError(fmt.Sprintf("Failed to initialize swarm: %s", output))
		}
		log.Infof("%s: swarm initialized succesfully", swarmLeader.Address)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader.Address)
	}

	mgrToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token manager -q"), exec.HideOutput())
	if err != nil {
		return NewError("failed to get swarm manager join-token")
	}
	config.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token worker -q"), exec.HideOutput())
	if err != nil {
		return NewError("failed to get swarm worker join-token")
	}
	config.WorkerJoinToken = workerToken
	return nil
}
