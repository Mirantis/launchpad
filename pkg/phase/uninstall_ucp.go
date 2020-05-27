package phase

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/swarm"
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

	image := fmt.Sprintf("%s/ucp:%s", config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	args := fmt.Sprintf("--id %s", swarm.ClusterID(swarmLeader))
	installCmd := swarmLeader.Configurer.DockerCommandf("run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s uninstall-ucp %s", image, args)
	err := swarmLeader.ExecCmd(installCmd, "", true, true)
	if err != nil {
		return NewError("Failed to run UCP uninstaller")
	}

	return nil
}
