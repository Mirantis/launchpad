package phase

import (
	"fmt"
	"time"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	"github.com/Mirantis/mcc/pkg/swarm"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// JoinWorkers phase implementation
type JoinWorkers struct {
	Analytics
	Dtr bool
}

// Title for the phase
func (p *JoinWorkers) Title() string {
	return "Join workers"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinWorkers) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	var hosts api.Hosts
	if p.Dtr {
		// If dtr roles are detected, add them to the list of workers
		hosts = config.Spec.WorkersAndDtrs()
	} else {
		hosts = config.Spec.Workers()
	}

	for _, h := range hosts {
		if swarm.IsSwarmNode(h) {
			log.Infof("%s: already a swarm node", h.Address)
			continue
		}
		joinCmd := h.Configurer.DockerCommandf("swarm join --token %s %s", config.WorkerJoinToken, swarmLeader.SwarmAddress())
		log.Debugf("%s: joining as worker", h.Address)
		err := h.ExecCmd(joinCmd, "", true, true)
		if err != nil {
			return NewError(fmt.Sprintf("Failed to join worker %s node to swarm", h.Address))
		}
		log.Infof("%s: joined succesfully", h.Address)
		if h.IsWindows() {
			// This is merely a workaround for the fact that we cannot reliably now detect if the connection is actually broken
			// with current ssh client config etc. the commands tried will timeout after several minutes only
			log.Infof("%s: wait for reconnect as swarm join on windows breaks existing connections", h.Address)
			// Wait for the swarm join actually break the connections and then reconnect.
			time.Sleep(5 * time.Second)
			err = retry.Do(
				func() error {
					h.Disconnect()
					err = h.Connect()
					if err != nil {
						return fmt.Errorf("error reconnecting host %s: %w", h.Address, err)
					}
					return nil
				})
			if err != nil {
				return err
			}
			log.Infof("%s: reconnected", h.Address)
		}
	}
	return nil
}
