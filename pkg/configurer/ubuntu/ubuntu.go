package ubuntu

import (
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

// InstallMKEBasePackages installs the needed base packages on Ubuntu
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	return c.InstallPackage(h, "curl", "apt-utils", "socat", "iputils-ping")
}

// UninstallMCR uninstalls docker-ee engine
func (c Configurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	err := h.Exec("sudo docker system prune -f")
	if err != nil {
		return err
	}
	if err := c.StopService(h, "docker"); err != nil {
		return err
	}
	if err := c.StopService(h, "containerd"); err != nil {
		return err
	}
	return h.Exec("sudo apt-get remove -y docker-ee docker-ee-cli && sudo apt autoremove -y")
}
