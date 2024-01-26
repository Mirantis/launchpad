package ubuntu

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/configurer"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	linux.Ubuntu
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages installs the needed base packages on Ubuntu.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "apt-utils", "socat", "iputils-ping"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}
	return nil
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h, c.Kind())
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

		if err := h.Exec("sudo apt-get remove -y docker-ee docker-ee-cli && sudo apt autoremove -y"); err != nil {
			return fmt.Errorf("failed to uninstall docker-ee apt package: %w", err)
		}
	}

	return nil
}
