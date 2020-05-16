package util

import (
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
)

// CollectUcpFacts gathers the current status of installed UCP setup
// Currently we only need to know the existing version and whether UCP is installed or not.
// In future we probably need more.
func CollectUcpFacts(swarmLeader *config.Host) (*config.UcpMetadata, error) {
	output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`inspect --format '{{ index .Config.Labels "com.docker.ucp.version"}}' ucp-proxy`))
	if err != nil {
		// We need to check the output to check if the container does not exist
		if strings.Contains(output, "No such object") {
			return &config.UcpMetadata{Installed: false}, nil
		}
		return nil, err
	}
	ucpMeta := &config.UcpMetadata{
		Installed:        true,
		InstalledVersion: output,
	}
	return ucpMeta, nil
}
