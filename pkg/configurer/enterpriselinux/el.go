package enterpriselinux

import (
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
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
func (c Configurer) UninstallMCR(h os.Host, _ string, engineConfig commonconfig.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h)
	if engineConfig.Prune {
		defer c.CleanupLingeringMCR(h, info)
	}
	if getDockerError == nil {
		if err := h.Exec("docker system prune -f"); err != nil {
			return fmt.Errorf("prune docker: %w", err)
		}

		if err := c.StopService(h, "docker"); err != nil {
			return fmt.Errorf("stop docker: %w", err)
		}

		if err := c.StopService(h, "containerd"); err != nil {
			return fmt.Errorf("stop containerd: %w", err)
		}

		if err := h.Exec("yum remove -y docker-ee docker-ee-cli", exec.Sudo(h)); err != nil {
			return fmt.Errorf("remove docker-ee yum package: %w", err)
		}
	}

	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h os.Host, scriptPath string, engineConfig commonconfig.MCRConfig) error {
	if isEC2 := c.isAWSInstance(h); !isEC2 {
		log.Debugf("%s: confirmed that this is not an AWS instance", h)
	} else if c.InstallPackage(h, "rh-amazon-rhui-client") == nil {
		log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", h)
	}

	if err := c.LinuxConfigurer.InstallMCR(h, scriptPath, engineConfig); err != nil {
		return fmt.Errorf("failed to install MCR: %w", err)
	}
	return nil
}

// function to check if the host is an AWS instance - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html
func (c Configurer) isAWSInstance(h os.Host) bool {
	found, err := h.ExecOutput("curl -s -m 5 http://169.254.169.254/latest/dynamic/instance-identity/document | grep region")
	if err != nil {
		log.Debugf("%s: curl on local-linked AWS id document failed: %v", h, err)
		return false
	}
	return strings.Contains(found, "region")
}
