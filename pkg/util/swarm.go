package util

import (
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// IsSwarmNode check whether the given node is already part of swarm
func IsSwarmNode(host *config.Host) bool {
	output, err := host.ExecWithOutput("sudo docker info --format '{{json .Swarm.NodeID}}'")
	if err != nil {
		log.Warnf("failed to get hosts swarm status")
		return false
	}

	if output == `""` {
		return false
	}

	return true
}

// SwarmClusterID digs the swarm cluster id from swarm leader host
func SwarmClusterID(leader *config.Host) string {
	output, err := leader.ExecWithOutput("sudo docker info --format '{{ .Swarm.Cluster.ID}}'")
	if err != nil {
		log.Warnf("failed to get hosts swarm status")
		return ""
	}

	return output
}
