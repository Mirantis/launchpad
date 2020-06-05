package phase

import (
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// UninstallUCP is the phase implementation for running UCP uninstall
type UninstallUCP struct {
	Analytics
}

// Title prints the phase title
func (p *UninstallUCP) Title() string {
	return "Uninstall UCP components"
}

// Run the installer container
func (p *UninstallUCP) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()
	if !config.Spec.Ucp.Metadata.Installed {
		log.Infof("%s: UCP is not installed, skipping", swarmLeader.Address)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	args := fmt.Sprintf("--id %s", swarm.ClusterID(swarmLeader))
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	uninstallCmd := swarmLeader.Configurer.DockerCommandf("run %s %s uninstall-ucp %s", strings.Join(runFlags, " "), image, args)
	err := swarmLeader.ExecCmd(uninstallCmd, "", true, true)
	if err != nil {
		return NewError("Failed to run UCP uninstaller")
	}

	return nil
}
