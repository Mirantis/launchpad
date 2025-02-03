package phase

import (
	"fmt"
	"time"

	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/swarm"
	retry "github.com/avast/retry-go"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// JoinWorkers phase implementation.
type JoinWorkers struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *JoinWorkers) Title() string {
	return "Join workers"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinWorkers) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()

	hosts := p.Config.Spec.WorkersAndMSRs()

	for _, h := range hosts {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", p.Config.Spec.MCR.Metadata.WorkerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as worker", h)
		err := h.Exec(joinCmd, exec.RedactString(p.Config.Spec.MCR.Metadata.WorkerJoinToken))
		if err != nil {
			return fmt.Errorf("failed to join worker %s node to swarm: %w", h, err)
		}
		log.Infof("%s: joined successfully", h)
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
				return fmt.Errorf("retry count exceeded: %w", err)
			}
			log.Infof("%s: reconnected", h)
		}
	}
	return nil
}
