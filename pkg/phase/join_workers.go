package phase

import (
	"fmt"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// JoinWorkers phase implementation
type JoinWorkers struct{}

// Title for the phase
func (p *JoinWorkers) Title() string {
	return "Join workers"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinWorkers) Run(config *config.ClusterConfig) error {
	start := time.Now()
	swarmLeader := config.Managers()[0]
	for _, h := range config.Workers() {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h.Address)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", config.WorkerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as worker", h.Address)
		err := h.Exec(joinCmd)
		if err != nil {
			return NewError(fmt.Sprintf("Failed to join worker %s node to swarm", h.Address))
		}
		log.Infof("%s: joined succesfully", h.Address)
	}
	duration := time.Since(start)
	props := analytics.NewAnalyticsEventProperties()
	props["duration"] = duration.Seconds()
	analytics.TrackEvent("Workers Joined", props)
	return nil
}
