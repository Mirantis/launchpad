package swarm

import (
	"fmt"
	"strings"

	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	log "github.com/sirupsen/logrus"
)

// IsSwarmNode check whether the given node is already part of swarm.
func IsSwarmNode(h *mkeconfig.Host) bool {
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
func NodeID(h *mkeconfig.Host) (string, error) {
	out, err := h.ExecOutput(h.Configurer.DockerCommandf(`info --format "{{.Swarm.NodeID}}"`))
	if err != nil {
		return "", fmt.Errorf("failed to get host's swarm node id: %w", err)
	}
	return out, nil
}

// ClusterID digs the swarm cluster id from swarm leader host.
func ClusterID(h *mkeconfig.Host) string {
	output, err := h.ExecOutput(h.Configurer.DockerCommandf(`info --format "{{ .Swarm.Cluster.ID}}"`))
	if err != nil {
		log.Warnf("%s: failed to get host's swarm status, probably not part of swarm", h)
		return ""
	}

	return output
}

// DefaultAddrPoolFallback is the overlay address pool Docker uses when a swarm
// was created without --default-addr-pool. Docker does not report a pool in that
// case, so this value is what such a cluster is actually running with.
const DefaultAddrPoolFallback = "10.0.0.0/8"

// DefaultAddrPool returns the overlay address pools in effect for the swarm the
// given host belongs to, or nil when the host is not part of a swarm.
//
// The pool is fixed when the swarm is created and cannot be changed afterwards:
// --default-addr-pool is a field of docker's swarm InitRequest but not of the
// Spec that "docker swarm update" mutates. Changing it requires dissolving and
// re-creating the swarm, which destroys every overlay network and service.
//
// A swarm created without the flag reports an empty pool list, in which case
// DefaultAddrPoolFallback is returned, so a non-empty result means "this host is
// in a swarm and these are the pools it allocates overlay networks from".
func DefaultAddrPool(h *mkeconfig.Host) ([]string, error) {
	// The swarm state is read in the same call so that "not in a swarm" is
	// distinguishable from "in a swarm with no explicit pool". Cluster is nil
	// outside a swarm and dereferencing it fails the whole template, hence the
	// guard.
	out, err := h.ExecOutput(h.Configurer.DockerCommandf(
		`info --format "{{.Swarm.LocalNodeState}}|{{if .Swarm.Cluster}}{{range .Swarm.Cluster.DefaultAddrPool}}{{.}} {{end}}{{end}}"`))
	if err != nil {
		return nil, fmt.Errorf("failed to get swarm overlay address pool: %w", err)
	}

	state, pools, _ := strings.Cut(out, "|")
	if strings.TrimSpace(state) != "active" {
		return nil, nil
	}

	if configured := strings.Fields(pools); len(configured) > 0 {
		return configured, nil
	}

	return []string{DefaultAddrPoolFallback}, nil
}
