package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

const configName string = "com.docker.ucp.config"

// InstallUCP is the phase implementation for running the actual UCP installer container
type InstallUCP struct{}

// Title prints the phase title
func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

// Run the installer container
func (p *InstallUCP) Run(config *config.ClusterConfig) error {
	swarmLeader := config.Controllers()[0]

	if config.Ucp.Metadata.Installed {
		log.Infof("UCP already installed at version %s, not running installer", config.Ucp.Metadata.InstalledVersion)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", config.Ucp.ImageRepo, config.Ucp.Version)

	installFlags := config.Ucp.InstallFlags
	if config.Ucp.ConfigData != "" {
		defer func() {
			err := swarmLeader.Execf("sudo docker config rm %s", configName)
			if err != nil {
				log.Warnf("Failed to remove the temporary UCP installer configuration %s : %s", configName, err)
			}
		}()

		installFlags = append(installFlags, "--existing-config")
		log.Info("Creating UCP configuration")
		err := swarmLeader.ExecCmd(fmt.Sprintf("sudo docker config create %s -", configName), config.Ucp.ConfigData)
		if err != nil {
			return err
		}
	}

	flags := strings.Join(installFlags, " ")
	installCmd := fmt.Sprintf("sudo docker run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install %s", image, flags)
	err := swarmLeader.Exec(installCmd)
	if err != nil {
		return NewError("Failed to run UCP installer")
	}
	return nil
}
