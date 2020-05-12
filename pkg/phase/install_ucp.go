package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

type InstallUCP struct{}

const InstallerImage = "docker/ucp:3.3.0-rc1"

func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

func (p *InstallUCP) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]

	// FIXME DO NOT USE HARDCODED PASSWD etc. :D
	installCmd := fmt.Sprintf("sudo docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install --admin-username admin --admin-password orcaorcaorca --force-minimums", InstallerImage)
	log.Debugf("Running installer with cmd: %s", installCmd)
	err := swarmLeader.Exec(installCmd)
	if err != nil {
		return fmt.Errorf("Failed to run UCP installer")
	}
	return nil
}
