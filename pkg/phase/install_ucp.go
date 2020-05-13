package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

type InstallUCP struct{}

func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

func (p *InstallUCP) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]
	image := fmt.Sprintf("%s/ucp:%s", config.Ucp.ImageRepo, config.Ucp.Version)
	flags := strings.Join(config.Ucp.InstallFlags, " ")
	installCmd := fmt.Sprintf("sudo docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install %s", image, flags)
	log.Debugf("Running installer with cmd: %s", installCmd)
	err := swarmLeader.Exec(installCmd)
	if err != nil {
		return fmt.Errorf("Failed to run UCP installer")
	}
	return nil
}
