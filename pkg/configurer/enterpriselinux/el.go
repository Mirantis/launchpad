package enterpriselinux

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/configurer"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
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
	if err := c.InstallPackage(h, "curl", "socat", "iptables", "iputils", "gzip"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}
	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, _ string, engineConfig common.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h)
	if engineConfig.Prune {
		defer c.CleanupLingeringMCR(h, info)
	}
	if getDockerError == nil {
		if err := h.Exec("sudo docker system prune -f"); err != nil {
			return fmt.Errorf("prune docker: %w", err)
		}

		if err := c.StopService(h, "docker"); err != nil {
			return fmt.Errorf("stop docker: %w", err)
		}

		if err := c.StopService(h, "containerd"); err != nil {
			return fmt.Errorf("stop containerd: %w", err)
		}

		if err := h.Exec("sudo yum remove -y docker-ee docker-ee-cli"); err != nil {
			return fmt.Errorf("remove docker-ee yum package: %w", err)
		}
	}

	return nil
}

// InstallMCR install Docker EE engine on Linux.
func (c Configurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	if h.Exec("sudo yum-config-manager --enable rhel-7-server-rhui-extras-rpms && sudo yum makecache fast") == nil {
		log.Infof("%s: enabled rhel-7-server-rhui-extras-rpms repository", h)
	}

	if err := c.LinuxConfigurer.InstallMCR(h, scriptPath, engineConfig); err != nil {
		return fmt.Errorf("failed to install MCR: %w", err)
	}
	return nil
}
