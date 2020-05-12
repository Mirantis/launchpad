package util

import (
	"github.com/Mirantis/mcc/pkg/host"
	log "github.com/sirupsen/logrus"
)

func IsSwarmNode(host *host.Host) bool {
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
