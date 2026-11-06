package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/swarm"
	"github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

// JoinManagers phase implementation.
type JoinManagers struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *JoinManagers) Title() string {
	return "Join managers to swarm"
}

// Run joins the manager nodes into swarm.
func (p *JoinManagers) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	for _, h := range p.Config.Spec.Managers() {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h)
			continue
		}
		// The join token stays in the command line: docker swarm join
		// accepts it only as --token, with no environment variable, file
		// or stdin variant, and --token "$(cat f)" would be expanded by
		// the shell before exec, putting the token in docker's own argv
		// anyway. See the same note in join_workers.go and PRODENG-3923.
		joinCmd := h.Configurer.DockerCmd().Raw("swarm join").Arg(fmt.Sprintf("--advertise-addr=%s", h.SwarmAddress())).Raw("--token").Arg(p.Config.Spec.MCR.Metadata.ManagerJoinToken).Arg(swarmLeader.SwarmAddress()).String()
		log.Debugf("%s: joining as manager", h)
		err := h.Exec(joinCmd, cmd.StreamOutput(), cmd.Redact(p.Config.Spec.MCR.Metadata.ManagerJoinToken))
		if err != nil {
			return fmt.Errorf("%s: failed to join manager node to swarm: %w", h, err)
		}
		log.Infof("%s: joined successfully", h)
	}
	return nil
}
