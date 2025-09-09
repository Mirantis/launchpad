package enterpriselinux

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/configurer"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
	log "github.com/sirupsen/logrus"
)

// Configurer is the EL family specific implementation of a host configurer.
type Configurer struct {
	linux.EnterpriseLinux
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages install all the needed base packages on the host.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "socat", "iptables", "iputils", "gzip", "openssh"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}
	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, _ string, engineConfig common.MCRConfig) error {
	if err := c.StopMCR(h, engineConfig); err != nil {
		return errors.Join(configurer.ErrorConfigurerMCRUninstall, err)
	}

	if err := c.uninstallMCRPackages(h); err != nil {
		return errors.Join(configurer.ErrorConfigurerMCRUninstall, err)
	}

	if err := c.uninstallMCRRepo(h); err != nil {
		return errors.Join(configurer.ErrorConfigurerMCRUninstall, err)
	}
	
	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	if err := c.installELAWSDependencies(h); err != nil {
		return errors.Join(configurer.ErrorConfigurerMCRInstall, err)
	}

	if err := c.installMCRRepo(h); err != nil {
		return errors.Join(configurer.ErrorConfigurerMCRInstall, err)

	}
	if err := c.installMCRPackages(h); err != nil {
		return errors.Join(configurer.ErrorConfigurerMCRInstall, err)		
	}

	return nil
}

// install the EL Repos
func (c Configurer) installMCRRepo(h os.Host) error {
	return nil
}

// un-install the EL Repos
func (c Configurer) uninstallMCRRepo(h os.Host) error {
	return nil
}

// install the MCR packages
func (c Configurer) installMCRPackages(h os.Host) error {
	return nil
}

// install the MCR packages
func (c Configurer) uninstallMCRPackages(h os.Host) error {
	if err := h.Exec("yum remove -y docker-ee docker-ee-cli", exec.Sudo(h)); err != nil {
		return fmt.Errorf("remove docker-ee yum package: %w", err)
	}
	return nil
}

// function to check if the host is an AWS instance - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (c Configurer) installELAWSDependencies(h os.Host) error {
	found, err := h.ExecOutput("curl -s -m 5 http://169.254.169.254/latest/dynamic/instance-identity/document | grep region")
	if err != nil {
		return err
	}

	// the test for an aws instance
	if strings.Contains(found, "region") {

		if err := c.InstallPackage(h, "rh-amazon-rhui-client"); err == nil {
			log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", h)
		} else {
			return fmt.Errorf("%s: appears to be an AWS EC2 instance, but failed to install rh-amazon-rhui-client: %w", h, err)
		}
	}

	return nil
}
