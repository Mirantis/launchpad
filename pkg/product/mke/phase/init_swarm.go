package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// InitSwarm phase implementation.
type InitSwarm struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *InitSwarm) Title() string {
	return "Initialize Swarm"
}

// Run initializes the swarm on the leader or skips if swarm is already initialized.
func (p *InitSwarm) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	if !swarm.IsSwarmNode(swarmLeader) {
		log.Infof("%s: initializing swarm", swarmLeader)
		err := swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("swarm init --advertise-addr=%s %s", swarmLeader.SwarmAddress(), p.Config.Spec.MCR.SwarmInstallFlags.Join()), exec.Redact(`--token \S+`))
		if err != nil {
			return fmt.Errorf("failed to initialize swarm: %w", err)
		}

		// Execute all swarm-post-init commands. These take care of
		// things like setting cert-expiry which cannot be done at the
		// time of swarm install.
		for _, swarmCmd := range p.Config.Spec.MCR.SwarmUpdateCommands {
			err := swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("%s", swarmCmd))
			if err != nil {
				return fmt.Errorf("post swarm init command (%s) failed: %w", swarmCmd, err)
			}
		}

		log.Infof("%s: swarm initialized successfully", swarmLeader)
	} else {
		log.Infof("%s: swarm already initialized", swarmLeader)
		if len(p.Config.Spec.MCR.SwarmInstallFlags) > 0 {
			log.Warnf("%s: swarm install flags ignored due to swarm cluster already existing", swarmLeader)
		}
	}

	mgrToken, err := swarmLeader.ExecOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token manager -q"), exec.HideOutput())
	if err != nil {
		return fmt.Errorf("%s: failed to get swarm manager join-token: %w", swarmLeader, err)
	}
	p.Config.Spec.MCR.Metadata.ManagerJoinToken = mgrToken

	workerToken, err := swarmLeader.ExecOutput(swarmLeader.Configurer.DockerCommandf("swarm join-token worker -q"), exec.HideOutput())
	if err != nil {
		return fmt.Errorf("%s: failed to get swarm worker join-token: %w", swarmLeader, err)
	}
	p.Config.Spec.MCR.Metadata.WorkerJoinToken = workerToken
	return nil
}
