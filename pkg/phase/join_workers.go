package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

type JoinWorkers struct{}

func (p *JoinWorkers) Title() string {
	return "Join workers"
}

func (p *JoinWorkers) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]
	for _, h := range config.Workers() {
		if util.IsSwarmNode(h) {
			log.Infof("%s: Already a swarm node, not gonna re-join as worker", h.Address)
			continue
		}
		joinCmd := fmt.Sprintf("sudo docker swarm join --token %s %s", config.WorkerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as worker", h.Address)
		err := h.Exec(joinCmd)
		if err != nil {
			return fmt.Errorf("Failed to join worker %s node to swarm", h.Address)
		}
		log.Infof("%s: joined succesfully", h.Address)
	}
	return nil
}
