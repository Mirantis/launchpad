package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// JoinManagers phase implementation
type JoinManagers struct{}

// Title for the phase
func (p *JoinManagers) Title() string {
	return "Join managers to swarm"
}

// Run joins the manager nodes into swarm
func (p *JoinManagers) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Managers()[0]
	for _, h := range config.Managers() {
		if util.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h.Address)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", config.ManagerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as manager", h.Address)
		output, err := h.ExecWithOutput(joinCmd)
		if err != nil {
			return NewError(fmt.Sprintf("Failed to join manager node to swarm: %s", output))
		}
		log.Infof("%s: joined succesfully", h.Address)
	}
	return nil
}
