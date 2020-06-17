package swarm

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	log "github.com/sirupsen/logrus"
)

// IsSwarmNode check whether the given node is already part of swarm
func IsSwarmNode(host *api.Host) bool {
	output, err := NodeID(host)
	if err != nil {
		log.Warnf("failed to get hosts swarm status")
		return false
	}

	if output == "" {
		return false
	}

	return true
}

// NodeID returns the hosts node id in swarm cluster
func NodeID(host *api.Host) (string, error) {
	return host.ExecWithOutput(host.Configurer.DockerCommandf(`info --format "{{.Swarm.NodeID}}"`))
}

// ClusterID digs the swarm cluster id from swarm leader host
func ClusterID(leader *api.Host) string {
	output, err := leader.ExecWithOutput(leader.Configurer.DockerCommandf(`info --format "{{ .Swarm.Cluster.ID}}"`))
	if err != nil {
		log.Warnf("failed to get host's swarm status, probably not part of swarm")
		return ""
	}

	return output
}
