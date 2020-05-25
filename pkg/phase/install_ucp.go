package phase

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
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
func (p *InstallUCP) Run(config *api.ClusterConfig) error {
	swarmLeader := config.Spec.SwarmLeader()

	if config.Spec.Ucp.Metadata.Installed {
		log.Infof("%s: UCP already installed at version %s, not running installer", swarmLeader.Address, config.Spec.Ucp.Metadata.InstalledVersion)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", config.Spec.Ucp.ImageRepo, config.Spec.Ucp.Version)
	installFlags := config.Spec.Ucp.InstallFlags
	if config.Spec.Ucp.ConfigData != "" {
		defer func() {
			err := swarmLeader.Execf("sudo docker config rm %s", configName)
			if err != nil {
				log.Warnf("Failed to remove the temporary UCP installer configuration %s : %s", configName, err)
			}
		}()

		installFlags = append(installFlags, "--existing-config")
		log.Info("Creating UCP configuration")
		configCmd := swarmLeader.Configurer.DockerCommandf("config create %s -", configName)
		err := swarmLeader.ExecCmd(configCmd, config.Spec.Ucp.ConfigData, false, false)
		if err != nil {
			return err
		}
	}

	if licenseFilePath := config.Spec.Ucp.LicenseFilePath; licenseFilePath != "" {
		log.Debugf("Installing with LicenseFilePath: %s", licenseFilePath)
		license, err := ioutil.ReadFile(licenseFilePath)
		if err != nil {
			return fmt.Errorf("error while reading license file %s: %v", licenseFilePath, err)
		}
		installFlags = append(installFlags, fmt.Sprintf("--license '%s'", string(license)))
	}
	flags := strings.Join(installFlags, " ")
	installCmd := swarmLeader.Configurer.DockerCommandf("run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install %s", image, flags)
	err := swarmLeader.ExecCmd(installCmd, "", true, true)
	if err != nil {
		return NewError("Failed to run UCP installer")
	}

	ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	config.Spec.Ucp.Metadata = ucpMeta
	config.State.ClusterID = swarm.ClusterID(swarmLeader)
	return nil
}
