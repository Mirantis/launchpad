package mcr

import (
	"errors"
	"fmt"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

var (
	ErrInvalidMCRConfig = errors.New("MCR configuration is invalid")
	ErrMCRNotRunning    = errors.New("MCR is not running")
)

// DrainNode drains a node from the workload via docker drain command.
func DrainNode(lead *mkeconfig.Host, h *mkeconfig.Host) error {
	nodeID, err := swarm.NodeID(h)
	if err != nil {
		return fmt.Errorf("failed to get node ID for %s: %w", h, err)
	}

	drainCmd := lead.Configurer.DockerCommandf("node update --availability drain %s", nodeID)
	if err := lead.Exec(drainCmd); err != nil {
		return fmt.Errorf("%s: failed to run MKE uninstaller: %w", lead, err)
	}
	if err := lead.Exec(drainCmd); err != nil {
		return fmt.Errorf("failed to drain node %s: %w", nodeID, err)
	}

	log.Infof("%s: node %s drained", lead, nodeID)
	return nil
}

// EnsureMCRRunning ensure that MCR is running.
func EnsureMCRRunning(h *mkeconfig.Host, _ commonconfig.MCRConfig) error {
	if _, err := h.MCRVersion(); err != nil {
		return fmt.Errorf("%w; %s", ErrMCRNotRunning, err.Error())
	}

	return nil
}
