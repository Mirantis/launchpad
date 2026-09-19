package phase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Mirantis/launchpad/pkg/phase"
	"github.com/Mirantis/launchpad/pkg/swarm"
	retry "github.com/avast/retry-go"
	"github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

// errNodeIDNotReady indicates a host's own docker engine has not yet
// reported a swarm NodeID for itself after joining -- see the comment on
// the confirmation retry in Run.
var errNodeIDNotReady = errors.New("host has not reported its swarm node id yet")

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
		joinCmd := h.Configurer.DockerCommandf("swarm join --advertise-addr=%s --token %s %s", h.SwarmAddress(), p.Config.Spec.MCR.Metadata.WorkerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as worker", h)
		err := h.Exec(joinCmd, cmd.Redact(p.Config.Spec.MCR.Metadata.WorkerJoinToken))
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
					err = h.Connect(context.Background())
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

		// `docker swarm join` returning success only means the join command
		// itself completed; it does not guarantee this host's own docker
		// engine has finished updating its local view of swarm state
		// (particularly right after the reconnect above, since joining
		// swarm on Windows tears down and re-establishes the connection).
		// Without this, a later phase's "docker info"/"docker node update"
		// can race this and see/use an empty NodeID -- observed in practice
		// as LabelNodes failing with `"docker node update" requires exactly
		// 1 argument` for a node whose NodeID query still came back empty.
		err = retry.Do(
			func() error {
				nodeID, nodeIDErr := swarm.NodeID(h)
				if nodeIDErr != nil {
					return fmt.Errorf("%s: %w", h, nodeIDErr)
				}
				if nodeID == "" {
					return fmt.Errorf("%s: %w", h, errNodeIDNotReady)
				}
				return nil
			},
			retry.Delay(time.Second*3),
			retry.Attempts(20),
		)
		if err != nil {
			return fmt.Errorf("failed to confirm swarm membership for %s: %w", h, err)
		}
	}
	return nil
}
