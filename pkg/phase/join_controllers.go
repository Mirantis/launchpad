package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// JoinManagers phase implementation
type JoinManagers struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *JoinManagers) Title() string {
	return "Join managers to swarm"
}

// Run joins the manager nodes into swarm
func (p *JoinManagers) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()

	for _, h := range p.config.Spec.Managers() {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h.Address)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", p.config.Spec.Ucp.Metadata.ManagerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as manager", h.Address)
		err := h.ExecCmd(joinCmd, "", true, true)
		if err != nil {
			return fmt.Errorf("%s: failed to join manager node to swarm: %s", h.Address, err.Error())
		}
		log.Infof("%s: joined successfully", h.Address)
	}
	return nil
}
