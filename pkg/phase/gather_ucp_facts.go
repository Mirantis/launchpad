package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"

	log "github.com/sirupsen/logrus"
)

type GatherUcpFacts struct{}

func (p *GatherUcpFacts) Title() string {
	return "Gather UCP facts"
}

func (p *GatherUcpFacts) Run(conf *config.ClusterConfig) error {
	swarmLeader := conf.Controllers()[0]
	exists, existingVersion, err := ucpExists(swarmLeader)

	if err != nil {
		return fmt.Errorf("Failed to check existing UCP version")
	}

	conf.Ucp.Metadata = &config.UcpMetadata{
		Installed:        exists,
		InstalledVersion: existingVersion,
	}

	log.Debugf("Found UCP facts: %+v", conf.Ucp.Metadata)

	return nil
}

// checks whether UCP is already running. If it is also returns the current version.
func ucpExists(swarmLeader *config.Host) (bool, string, error) {
	output, err := swarmLeader.ExecWithOutput(`sudo docker inspect --format '{{ index .Config.Labels "com.docker.ucp.version"}}' ucp-proxy`)
	if err != nil {
		// We need to check the output to check if the container does not exist
		if strings.Contains(output, "No such object") {
			return false, "", nil
		}
		return false, "", err
	}
	return true, output, nil
}
