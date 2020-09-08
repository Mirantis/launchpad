package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// JoinManagers phase implementation
type JoinManagers struct {
	Analytics
}

// Title for the phase
func (p *JoinManagers) Title() string {
	return "Join managers to swarm"
}

// Run joins the manager nodes into swarm
func (p *JoinManagers) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	for _, h := range config.Spec.Managers() {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h.Address)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", config.Spec.Ucp.Metadata.ManagerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as manager", h.Address)
		err := h.ExecCmd(joinCmd, "", true, true)
		if err != nil {
			return NewError(fmt.Sprintf("Failed to join manager node to swarm"))
		}
		log.Infof("%s: joined successfully", h.Address)
	}
	return nil
}
