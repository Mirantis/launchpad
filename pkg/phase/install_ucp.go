package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/ucp"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

const configName string = "com.docker.ucp.config"

// InstallUCP is the phase implementation for running the actual UCP installer container
type InstallUCP struct {
	Analytics
	BasicPhase
	SkipCleanup bool
}

// Title prints the phase title
func (p *InstallUCP) Title() string {
	return "Install UCP components"
}

// Run the installer container
func (p *InstallUCP) Run() (err error) {
	swarmLeader := p.config.Spec.SwarmLeader()

	if !p.SkipCleanup {
		defer func() {
			if err != nil {
				log.Println("Cleaning-up")
				if cleanupErr := cleanupUcp(swarmLeader); cleanupErr != nil {
					log.Warnln("Error while cleaning-up resources")
				}
			}
		}()
	}

	p.EventProperties = map[string]interface{}{
		"ucp_version": p.config.Spec.Ucp.Version,
	}

	if p.config.Spec.Ucp.Metadata.Installed {
		log.Infof("%s: UCP already installed at version %s, not running installer", swarmLeader, p.config.Spec.Ucp.Metadata.InstalledVersion)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", p.config.Spec.Ucp.ImageRepo, p.config.Spec.Ucp.Version)
	installFlags := p.config.Spec.Ucp.InstallFlags

	if p.config.Spec.Ucp.CACertData != "" && p.config.Spec.Ucp.CertData != "" && p.config.Spec.Ucp.KeyData != "" {
		err := p.installCertificates(p.config)
		if err != nil {
			return err
		}
		installFlags = append(installFlags, " --external-server-cert")
	}

	if p.config.Spec.Ucp.ConfigData != "" {
		defer func() {
			err := swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("config rm %s", configName))
			if err != nil {
				log.Warnf("Failed to remove the temporary UCP installer configuration %s : %s", configName, err)
			}
		}()

		installFlags = append(installFlags, "--existing-config")
		log.Info("Creating UCP configuration")
		configCmd := swarmLeader.Configurer.DockerCommandf("config create %s -", configName)
		err := swarmLeader.Exec(configCmd, exec.Stdin(p.config.Spec.Ucp.ConfigData))
		if err != nil {
			return err
		}
	}

	if licenseFilePath := p.config.Spec.Ucp.LicenseFilePath; licenseFilePath != "" {
		log.Debugf("Installing UCP with LicenseFilePath: %s", licenseFilePath)
		licenseFlag, err := util.SetupLicenseFile(p.config.Spec.Ucp.LicenseFilePath)
		if err != nil {
			return fmt.Errorf("error while reading license file %s: %v", licenseFilePath, err)
		}
		installFlags = append(installFlags, licenseFlag)
	}

	if p.config.Spec.Ucp.Cloud != nil {
		if p.config.Spec.Ucp.Cloud.Provider != "" {
			installFlags = append(installFlags, fmt.Sprintf("--cloud-provider %s", p.config.Spec.Ucp.Cloud.Provider))
		}
		if p.config.Spec.Ucp.Cloud.ConfigData != "" {
			applyCloudConfig(p.config)
		}
	}

	if api.IsCustomImageRepo(p.config.Spec.Ucp.ImageRepo) {
		// In case of custom repo, don't let UCP check the images
		installFlags = append(installFlags, "--pull never")
	}
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}

	installCmd := swarmLeader.Configurer.DockerCommandf("run %s %s install %s", strings.Join(runFlags, " "), image, strings.Join(installFlags, " "))
	err = swarmLeader.Exec(installCmd, exec.StreamOutput(), exec.Redact(`--admin\S+`))
	if err != nil {
		return fmt.Errorf("%s: failed to run UCP installer: %s", swarmLeader, err.Error())
	}

	err = ucp.CollectFacts(swarmLeader, p.config.Spec.Ucp.Metadata)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing UCP details: %s", swarmLeader, err.Error())
	}

	return nil
}

// installCertificates installs user supplied UCP certificates
func (p *InstallUCP) installCertificates(config *api.ClusterConfig) error {
	log.Infof("Installing UCP certificates")
	managers := config.Spec.Managers()
	err := managers.ParallelEach(func(h *api.Host) error {
		err := h.Exec(h.Configurer.DockerCommandf("volume inspect ucp-controller-server-certs"))
		if err != nil {
			log.Infof("%s: creating ucp-controller-server-certs volume", h)
			err := h.Exec(h.Configurer.DockerCommandf("volume create ucp-controller-server-certs"))
			if err != nil {
				return err
			}
		}

		dir, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`volume inspect ucp-controller-server-certs --format "{{ .Mountpoint }}"`))
		if err != nil {
			return err
		}

		log.Infof("%s: installing certificate files to %s", h, dir)
		err = h.Configurer.WriteFile(fmt.Sprintf("%s/ca.pem", dir), config.Spec.Ucp.CACertData, "0600")
		if err != nil {
			return err
		}
		err = h.Configurer.WriteFile(fmt.Sprintf("%s/cert.pem", dir), config.Spec.Ucp.CertData, "0600")
		if err != nil {
			return err
		}
		err = h.Configurer.WriteFile(fmt.Sprintf("%s/key.pem", dir), config.Spec.Ucp.KeyData, "0600")
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func applyCloudConfig(config *api.ClusterConfig) error {
	configData := config.Spec.Ucp.Cloud.ConfigData
	provider := config.Spec.Ucp.Cloud.Provider

	var destFile string
	if provider == "azure" {
		destFile = "/etc/kubernetes/azure.json"
	} else if provider == "openstack" {
		destFile = "/etc/kubernetes/openstack.conf"
	} else {
		return fmt.Errorf("Spec.Cloud.configData is only supported with Azure and OpenStack cloud providers")
	}

	err := runParallelOnHosts(config.Spec.Hosts, config, func(h *api.Host, c *api.ClusterConfig) error {
		log.Infof("%s: copying cloud provider (%s) config to %s", h, provider, destFile)
		return h.Configurer.WriteFile(destFile, configData, "0700")
	})

	return err
}

func cleanupUcp(h *api.Host) error {
	containersToRemove, err := h.ExecWithOutput(h.Configurer.DockerCommandf("ps -aq --filter name=ucp-"))
	if err != nil {
		return err
	}
	if strings.Trim(containersToRemove, " ") == "" {
		log.Debugf("No containers to remove")
		return nil
	}
	containersToRemove = strings.ReplaceAll(containersToRemove, "\n", " ")
	if err := h.Exec(h.Configurer.DockerCommandf("rm -f %s", containersToRemove)); err != nil {
		return err
	}

	return nil
}
