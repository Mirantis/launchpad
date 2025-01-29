package swarm

import (
	"fmt"

	github.com/Mirantis/launchpad/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// IsSwarmNode check whether the given node is already part of swarm.
func IsSwarmNode(h *api.Host) bool {
	output, err := NodeID(h)
	if err != nil {
		log.Warnf("%s: failed to get host's swarm status", h)
		return false
	}

	if output == "" {
		return false
	}

	return true
}

// NodeID returns the hosts node id in swarm cluster.
func NodeID(h *api.Host) (string, error) {
	out, err := h.ExecOutput(h.Configurer.DockerCommandf(`info --format "{{.Swarm.NodeID}}"`))
	if err != nil {
		return "", fmt.Errorf("failed to get host's swarm node id: %w", err)
	}
	return out, nil
}

// ClusterID digs the swarm cluster id from swarm leader host.
func ClusterID(h *api.Host) string {
	output, err := h.ExecOutput(h.Configurer.DockerCommandf(`info --format "{{ .Swarm.Cluster.ID}}"`))
	if err != nil {
		log.Warnf("%s: failed to get host's swarm status, probably not part of swarm", h)
		return ""
	}

	return output
}
