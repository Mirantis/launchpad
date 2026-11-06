package phase

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	mcclog "github.com/Mirantis/launchpad/pkg/log"
	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/util/installutil"
	"github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

const configName string = "com.docker.ucp.config"

// InstallMKE is the phase implementation for running the actual MKE installer container.
type InstallMKE struct {
	phase.Analytics
	phase.BasicPhase
	phase.CleanupDisabling

	leader *mkeconfig.Host
}

// Title prints the phase title.
func (p *InstallMKE) Title() string {
	return "Install MKE components"
}

// Run the installer container.
func (p *InstallMKE) Run() error {
	p.leader = p.Config.Spec.SwarmLeader()

	p.EventProperties = map[string]interface{}{
		"mke_version": p.Config.Spec.MKE.Version,
	}

	if p.Config.Spec.MKE.Metadata.Installed {
		log.Infof("%s: MKE already installed at version %s, not running installer", p.leader, p.Config.Spec.MKE.Metadata.InstalledVersion)
		return nil
	}

	installFlags := p.Config.Spec.MKE.InstallFlags

	if mcclog.Debug {
		installFlags.AddUnlessExist("--debug")
	}

	if p.Config.Spec.MKE.ConfigData != "" {
		defer func() {
			err := p.leader.Exec(p.leader.Configurer.DockerCmd().Raw("config rm").Arg(configName).String())
			if err != nil {
				log.Warnf("Failed to remove the temporary MKE installer configuration %s : %s", configName, err)
			}
		}()

		installFlags.AddUnlessExist("--existing-config")
		log.Info("Creating MKE configuration")
		configCmd := p.leader.Configurer.DockerCmd().Raw("config create").Arg(configName).Raw("-").String()
		err := p.leader.Exec(configCmd, cmd.StdinString(p.Config.Spec.MKE.ConfigData))
		if err != nil {
			return fmt.Errorf("%s: failed to create MKE configuration: %w", p.leader, err)
		}
	}

	// Secrets go to the installer as environment values, staged in
	// mode-0600 files on the host by pkg/util/secretenv, so they are absent
	// from the remote command line and from everything that records it.
	secretEnv := map[string]string{
		"UCP_ADMIN_USER":     p.Config.Spec.MKE.AdminUsername,
		"UCP_ADMIN_PASSWORD": p.Config.Spec.MKE.AdminPassword,
	}

	if licenseFilePath := p.Config.Spec.MKE.LicenseFilePath; licenseFilePath != "" {
		log.Debugf("Installing MKE with LicenseFilePath: %s", licenseFilePath)
		license, err := installutil.ReadLicense(licenseFilePath)
		if err != nil {
			return fmt.Errorf("error while reading license file %s: %w", licenseFilePath, err)
		}
		secretEnv["UCP_LICENSE"] = license
	}

	if p.Config.Spec.MKE.Cloud != nil {
		if p.Config.Spec.MKE.Cloud.Provider != "" {
			installFlags.AddUnlessExist("--cloud-provider " + p.Config.Spec.MKE.Cloud.Provider)
		}
		if p.Config.Spec.MKE.Cloud.ConfigData != "" {
			if err := applyCloudConfig(p.Config); err != nil {
				return err
			}
		}
	}

	if mkeconfig.IsCustomImageRepo(p.Config.Spec.MKE.ImageRepo) {
		// In case of custom repo, don't let MKE check the images
		installFlags.AddUnlessExist("--pull never")
	}

	log.Debugf("%s: install flags: %s", p.leader, installFlags.Join())
	// redactOptions, not a hand written list: the username is a prefix of
	// many passwords and rig masks in registration order.
	execOpts := append([]cmd.ExecOption{cmd.StreamOutput()},
		redactOptions(p.Config.Spec.MKE.AdminUsername, p.Config.Spec.MKE.AdminPassword)...)
	output, err := mke.Bootstrap("install", *p.Config, mke.BootstrapOptions{
		OperationFlags:  installFlags,
		CleanupDisabled: p.CleanupDisabled(),
		SecretEnv:       secretEnv,
		ExecOptions:     execOpts,
	})
	if err != nil {
		return fmt.Errorf("%s: failed to run MKE installer: \n output: %s \n error: %w", p.leader, output, err)
	}

	err = mke.CollectFacts(p.leader, p.Config.Spec.MKE.Metadata)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MKE details: %w", p.leader, err)
	}

	return nil
}

var errUnsupportedProvider = errors.New("unsupported cloud provider")

func applyCloudConfig(config *mkeconfig.ClusterConfig) error {
	configData := config.Spec.MKE.Cloud.ConfigData
	provider := config.Spec.MKE.Cloud.Provider

	var destFile string
	switch provider {
	case "azure":
		destFile = "/etc/kubernetes/azure.json"
	case "openstack":
		destFile = "/etc/kubernetes/openstack.conf"
	default:
		return fmt.Errorf("%w: spec.Cloud.configData is only supported with Azure and OpenStack cloud providers", errUnsupportedProvider)
	}

	err := phase.RunParallelOnHosts(config.Spec.Hosts, config, func(h *mkeconfig.Host, _ *mkeconfig.ClusterConfig) error {
		if h.IsWindows() {
			log.Warnf("%s: cloud provider configuration is not suppported on windows", h)
			return nil
		}

		log.Infof("%s: copying cloud provider (%s) config to %s", h, provider, destFile)
		if err := h.Sudo().FS().WriteFile(destFile, []byte(configData), fs.FileMode(0o600)); err != nil {
			return fmt.Errorf("%s: failed to write cloud provider config: %w", h, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to apply cloud provider config: %w", err)
	}
	return nil
}

func cleanupmke(h *mkeconfig.Host) error {
	containersToRemove, err := h.ExecOutput(h.Configurer.DockerCommandf("ps -aq --filter name=ucp-"))
	if err != nil {
		return fmt.Errorf("%s: failed to list mke containers: %w", h, err)
	}
	if strings.Trim(containersToRemove, " ") == "" {
		log.Debugf("No containers to remove")
		return nil
	}
	if err := h.Exec(h.Configurer.DockerCmd().Raw("rm -f").Args(strings.Fields(containersToRemove)...).String()); err != nil {
		return fmt.Errorf("%s: failed to remove mke containers: %w", h, err)
	}

	return nil
}

// CleanUp removes ucp containers after a failed installation.
func (p *InstallMKE) CleanUp() {
	log.Infof("Cleaning up for '%s'", p.Title())
	if err := cleanupmke(p.leader); err != nil {
		log.Warnf("Error while cleaning-up resources for '%s': %s", p.Title(), err.Error())
	}
}
