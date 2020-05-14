package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

type JoinControllers struct{}

func (p *JoinControllers) Title() string {
	return "Join controllers"
}

func (p *JoinControllers) Run(config *config.ClusterConfig) *PhaseError {
	swarmLeader := config.Controllers()[0]
	for _, h := range config.Controllers() {
		if util.IsSwarmNode(h) {
			log.Infof("%s: Already a swarm node, not gonna re-join as manager", h.Address)
			continue
		}
		joinCmd := fmt.Sprintf("sudo docker swarm join --token %s %s", config.ManagerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as controller", h.Address)
		output, err := h.ExecWithOutput(joinCmd)
		if err != nil {
			return NewPhaseError(fmt.Sprintf("Failed to join controller node to swarm: %s", output))
		}
		log.Debugf("%s: joined succesfully", h.Address)
	}
	return nil
}
