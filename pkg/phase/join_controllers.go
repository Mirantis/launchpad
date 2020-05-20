package phase

import (
	"fmt"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/swarm"
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
	start := time.Now()
	swarmLeader := config.Managers()[0]
	for _, h := range config.Managers() {
		if swarm.IsSwarmNode(h) {
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
	duration := time.Since(start)
	props := analytics.NewAnalyticsEventProperties()
	props["duration"] = duration.Seconds()
	analytics.TrackEvent("Managers Joined", props)
	return nil
}
