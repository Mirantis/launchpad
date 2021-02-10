package phase

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

const configName string = "com.docker.ucp.config"

// InstallMKE is the phase implementation for running the actual MKE installer container
type InstallMKE struct {
	phase.Analytics
	phase.BasicPhase
	SkipCleanup bool
}

// Title prints the phase title
func (p *InstallMKE) Title() string {
	return "Install MKE components"
}

// Run the installer container
func (p *InstallMKE) Run() (err error) {
	swarmLeader := p.Config.Spec.SwarmLeader()

	if !p.SkipCleanup {
		defer func() {
			if err != nil {
				log.Println("Cleaning-up")
				if cleanupErr := cleanupmke(swarmLeader); cleanupErr != nil {
					log.Warnln("Error while cleaning-up resources")
				}
			}
		}()
	}

	p.EventProperties = map[string]interface{}{
		"mke_version": p.Config.Spec.MKE.Version,
	}

	if p.Config.Spec.MKE.Metadata.Installed {
		log.Infof("%s: MKE already installed at version %s, not running installer", swarmLeader, p.Config.Spec.MKE.Metadata.InstalledVersion)
		return nil
	}

	image := p.Config.Spec.MKE.GetBootstrapperImage()
	installFlags := p.Config.Spec.MKE.InstallFlags

	if p.Config.Spec.MKE.CACertData != "" && p.Config.Spec.MKE.CertData != "" && p.Config.Spec.MKE.KeyData != "" {
		err := p.installCertificates(p.Config)
		if err != nil {
			return err
		}
		installFlags.AddUnlessExist("--external-server-cert")
	}

	if p.Config.Spec.MKE.ConfigData != "" {
		defer func() {
			err := swarmLeader.Exec(swarmLeader.Configurer.DockerCommandf("config rm %s", configName))
			if err != nil {
				log.Warnf("Failed to remove the temporary MKE installer configuration %s : %s", configName, err)
			}
		}()

		installFlags.AddUnlessExist("--existing-config")
		log.Info("Creating MKE configuration")
		configCmd := swarmLeader.Configurer.DockerCommandf("config create %s -", configName)
		err := swarmLeader.Exec(configCmd, exec.Stdin(p.Config.Spec.MKE.ConfigData))
		if err != nil {
			return err
		}
	}

	if licenseFilePath := p.Config.Spec.MKE.LicenseFilePath; licenseFilePath != "" {
		log.Debugf("Installing MKE with LicenseFilePath: %s", licenseFilePath)
		licenseFlag, err := util.SetupLicenseFile(p.Config.Spec.MKE.LicenseFilePath)
		if err != nil {
			return fmt.Errorf("error while reading license file %s: %v", licenseFilePath, err)
		}
		installFlags.AddUnlessExist(licenseFlag)
	}

	if p.Config.Spec.MKE.Cloud != nil {
		if p.Config.Spec.MKE.Cloud.Provider != "" {
			installFlags.AddUnlessExist("--cloud-provider " + p.Config.Spec.MKE.Cloud.Provider)
		}
		if p.Config.Spec.MKE.Cloud.ConfigData != "" {
			applyCloudConfig(p.Config)
		}
	}

	if api.IsCustomImageRepo(p.Config.Spec.MKE.ImageRepo) {
		// In case of custom repo, don't let MKE check the images
		installFlags.AddUnlessExist("--pull never")
	}
	runFlags := common.Flags{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled(swarmLeader) {
		runFlags.Add("--security-opt label=disable")
	}

	if p.Config.Spec.MKE.AdminUsername != "" {
		installFlags.AddUnlessExist("--admin-username " + p.Config.Spec.MKE.AdminUsername)
	}

	if p.Config.Spec.MKE.AdminPassword != "" {
		installFlags.AddUnlessExist("--admin-password " + p.Config.Spec.MKE.AdminPassword)
	}

	installCmd := swarmLeader.Configurer.DockerCommandf("run %s %s install %s", runFlags.Join(), image, installFlags.Join())
	output, err := swarmLeader.ExecOutput(installCmd, exec.StreamOutput(), exec.RedactString(p.Config.Spec.MKE.AdminUsername, p.Config.Spec.MKE.AdminPassword))
	if err != nil {
		return fmt.Errorf("%s: failed to run MKE installer: %s", swarmLeader, err.Error())
	}

	if installFlags.GetValue("--admin-password") == "" {
		re := regexp.MustCompile(`msg="Generated random admin password: (.+?)"`)
		md := re.FindStringSubmatch(output)
		if len(md) > 0 && md[1] != "" {
			log.Warnf("Using an automatically generated password for MKE admin user: %s -- you will have to set it to Spec.MKE.AdminPassword for any subsequent launchpad runs.", md[1])
			p.Config.Spec.MKE.AdminPassword = md[1]
			if p.Config.Spec.MKE.AdminUsername == "" {
				log.Debugf("defaulting to mke admin username 'admin'")
				p.Config.Spec.MKE.AdminUsername = "admin"
			}
		}
	}

	err = mke.CollectFacts(swarmLeader, p.Config.Spec.MKE.Metadata)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MKE details: %s", swarmLeader, err.Error())
	}

	return nil
}

// installCertificates installs user supplied MKE certificates
func (p *InstallMKE) installCertificates(config *api.ClusterConfig) error {
	log.Infof("Installing MKE certificates")
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

		dir, err := h.ExecOutput(h.Configurer.DockerCommandf(`volume inspect ucp-controller-server-certs --format "{{ .Mountpoint }}"`))
		if err != nil {
			return err
		}

		log.Infof("%s: installing certificate files to %s", h, dir)
		err = h.Configurer.WriteFile(h, fmt.Sprintf("%s/ca.pem", dir), config.Spec.MKE.CACertData, "0600")
		if err != nil {
			return err
		}
		err = h.Configurer.WriteFile(h, fmt.Sprintf("%s/cert.pem", dir), config.Spec.MKE.CertData, "0600")
		if err != nil {
			return err
		}
		err = h.Configurer.WriteFile(h, fmt.Sprintf("%s/key.pem", dir), config.Spec.MKE.KeyData, "0600")
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func applyCloudConfig(config *api.ClusterConfig) error {
	configData := config.Spec.MKE.Cloud.ConfigData
	provider := config.Spec.MKE.Cloud.Provider

	var destFile string
	if provider == "azure" {
		destFile = "/etc/kubernetes/azure.json"
	} else if provider == "openstack" {
		destFile = "/etc/kubernetes/openstack.conf"
	} else {
		return fmt.Errorf("Spec.Cloud.configData is only supported with Azure and OpenStack cloud providers")
	}

	err := phase.RunParallelOnHosts(config.Spec.Hosts, config, func(h *api.Host, c *api.ClusterConfig) error {
		log.Infof("%s: copying cloud provider (%s) config to %s", h, provider, destFile)
		return h.Configurer.WriteFile(h, destFile, configData, "0700")
	})

	return err
}

func cleanupmke(h *api.Host) error {
	containersToRemove, err := h.ExecOutput(h.Configurer.DockerCommandf("ps -aq --filter name=ucp-"))
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
