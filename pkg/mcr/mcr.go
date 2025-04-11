package mcr

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/Mirantis/launchpad/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// EnsureMCRVersion ensure that MCR is running after install/upgrade, and update the host information
// @NOTE will reboot the machine if MCR isn't detected
// @SEE PRODENG-2789 : we no longer perform version checks, as the MCR versions don't always match the spec string.
func EnsureMCRVersion(h *api.Host, specMcrVersion string) error {
	currentVersion, err := h.MCRVersion()
	if err != nil {
		if err := h.Reboot(); err != nil {
			return fmt.Errorf("%s: failed to reboot after container runtime installation: %w", h, err)
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after installation: %w", h, err)
		}
		// as we rebooted the machine, no need to also restart MCR
		h.Metadata.MCRRestartRequired = false
	}

	log.Infof("%s: MCR version %s (requested: %s)", h, currentVersion, specMcrVersion)
	h.Metadata.MCRVersion = currentVersion

	return nil
}

// DrainNode drains a node from the workload via docker drain command.
func DrainNode(lead *api.Host, h *api.Host) error {
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
