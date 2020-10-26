package phase

import (
	"fmt"
	"time"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/swarm"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// JoinWorkers phase implementation
type JoinWorkers struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *JoinWorkers) Title() string {
	return "Join workers"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinWorkers) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()

	hosts := p.config.Spec.WorkersAndDtrs()

	for _, h := range hosts {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", p.config.Spec.Ucp.Metadata.WorkerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as worker", h)
		err := h.Exec(joinCmd, exec.Redact(p.config.Spec.Ucp.Metadata.WorkerJoinToken))
		if err != nil {
			return fmt.Errorf("Failed to join worker %s node to swarm", h)
		}
		log.Infof("%s: joined succesfully", h)
		if h.IsWindows() {
			// This is merely a workaround for the fact that we cannot reliably now detect if the connection is actually broken
			// with current ssh client config etc. the commands tried will timeout after several minutes only
			log.Infof("%s: wait for reconnect as swarm join on windows breaks existing connections", h)
			// Wait for the swarm join actually break the connections and then reconnect.
			time.Sleep(5 * time.Second)
			err = retry.Do(
				func() error {
					h.Disconnect()
					err = h.Connect()
					if err != nil {
						return fmt.Errorf("error reconnecting host %s: %w", h, err)
					}
					return nil
				})
			if err != nil {
				return err
			}
			log.Infof("%s: reconnected", h)
		}
	}
	return nil
}
