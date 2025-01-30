package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/swarm"
	"github.com/k0sproject/rig/exec"
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
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", p.Config.Spec.MCR.Metadata.ManagerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as manager", h)
		err := h.Exec(joinCmd, exec.StreamOutput(), exec.RedactString(p.Config.Spec.MCR.Metadata.ManagerJoinToken))
		if err != nil {
			return fmt.Errorf("%s: failed to join manager node to swarm: %w", h, err)
		}
		log.Infof("%s: joined successfully", h)
	}
	return nil
}
