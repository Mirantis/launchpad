package swarm

import (
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// IsSwarmNode check whether the given node is already part of swarm
func IsSwarmNode(host *config.Host) bool {
	output, err := host.ExecWithOutput(host.Configurer.DockerCommandf(`info --format "{{.Swarm.NodeID}}"`))
	if err != nil {
		log.Warnf("failed to get hosts swarm status")
		return false
	}

	if output == "" {
		return false
	}

	return true
}

// ClusterID digs the swarm cluster id from swarm leader host
func ClusterID(leader *config.Host) string {
	output, err := leader.ExecWithOutput(leader.Configurer.DockerCommandf(`info --format "{{ .Swarm.Cluster.ID}}"`))
	if err != nil {
		log.Warnf("failed to get host's swarm status, probably not part of swarm")
		return ""
	}

	return output
}
