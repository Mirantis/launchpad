package phase

import (
	"fmt"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/Mirantis/mcc/pkg/ucp"

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
	start := time.Now()
	swarmLeader := config.Managers()[0]

	if config.Ucp.Metadata.Installed {
		log.Infof("%s: UCP already installed at version %s, not running installer", swarmLeader.Address, config.Ucp.Metadata.InstalledVersion)
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
		configCmd := swarmLeader.Configurer.DockerCommandf("config create %s -", configName)
		err := swarmLeader.ExecCmd(configCmd, config.Ucp.ConfigData, false)
		if err != nil {
			return err
		}
	}

	flags := strings.Join(installFlags, " ")
	installCmd := swarmLeader.Configurer.DockerCommandf("run --rm -i -v /var/run/docker.sock:/var/run/docker.sock %s install %s", image, flags)
	err := swarmLeader.ExecCmd(installCmd, "", true)
	if err != nil {
		return NewError("Failed to run UCP installer")
	}

	ucpMeta, err := ucp.CollectUcpFacts(swarmLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader.Address, err.Error())
	}
	config.Ucp.Metadata = ucpMeta
	config.State.ClusterID = swarm.ClusterID(swarmLeader)
	duration := time.Since(start)
	props := analytics.NewAnalyticsEventProperties()
	props["duration"] = duration.Seconds()
	props["ucp_version"] = config.Ucp.Version
	analytics.TrackEvent("UCP Installed", props)
	return nil
}
