package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// JoinManagers phase implementation
type JoinManagers struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *JoinManagers) Title() string {
	return "Join managers to swarm"
}

// Run joins the manager nodes into swarm
func (p *JoinManagers) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	for _, h := range p.Config.Spec.Managers() {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", p.Config.Spec.Ucp.Metadata.ManagerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as manager", h)
		err := h.Exec(joinCmd, exec.StreamOutput(), exec.RedactString(p.Config.Spec.Ucp.Metadata.ManagerJoinToken))
		if err != nil {
			return fmt.Errorf("%s: failed to join manager node to swarm: %s", h, err.Error())
		}
		log.Infof("%s: joined successfully", h)
	}
	return nil
}
