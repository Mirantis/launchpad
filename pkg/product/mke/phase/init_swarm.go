package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/swarm"

	log "github.com/sirupsen/logrus"
)

// InitSwarm phase implementation
type InitSwarm struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *InitSwarm) Title() string {
	return "Initialize Swarm"
}

// Run initializes the swarm on the leader or skips if swarm is already initialized
func (p *InitSwarm) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	if !swarm.IsSwarmNode(swarmLeader) {
		log.Infof("%s: initializing swarm", swarmLeader)
		output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm init --advertise-addr=%s", swarmLeader.SwarmAddress()), exec.Redact(`--token \S+`))
		if err != nil {
			return fmt.Errorf("failed to initialize swarm: %s", output)
		}
		log.Infof("%s: swarm initialized successfully", swarmLeader)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader)
	}

	mgrToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token manager -q"), exec.HideOutput())
	if err != nil {
		return fmt.Errorf("%s: failed to get swarm manager join-token: %s", swarmLeader, err.Error())
	}
	p.Config.Spec.MKE.Metadata.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token worker -q"), exec.HideOutput())
	if err != nil {
		return fmt.Errorf("%s: failed to get swarm worker join-token: %s", swarmLeader, err.Error())
	}
	p.Config.Spec.MKE.Metadata.WorkerJoinToken = workerToken
	return nil
}
