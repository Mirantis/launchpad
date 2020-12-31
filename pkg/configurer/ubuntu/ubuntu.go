package ubuntu

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/resolver"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages installs the needed base packages on Ubuntu
func (c *Configurer) InstallMKEBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q curl apt-utils socat iputils-ping")
}

// InstallK0sBasePackages installs the needed base packages on Ubuntu
func (c *Configurer) InstallK0sBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q curl")
}

// UninstallMCR uninstalls docker-ee engine
func (c *Configurer) UninstallMCR(scriptPath string, engineConfig common.MCRConfig) error {
	err := c.Host.Exec("sudo docker system prune -f")
	if err != nil {
		return err
	}
	err = c.Host.Exec("sudo systemctl stop docker")
	if err != nil {
		return err
	}
	err = c.Host.Exec("sudo systemctl stop containerd")
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo apt-get remove -y docker-ee docker-ee-cli && sudo apt autoremove -y")
}

func resolveUbuntuConfigurer(h configurer.Host, os *common.OsRelease) interface{} {
	if os.ID != "ubuntu" {
		return nil
	}
	switch os.Version {
	case "18.04":
		configurer := &BionicConfigurer{
			Configurer: Configurer{
				LinuxConfigurer: configurer.LinuxConfigurer{
					Host: h,
				},
			},
		}
		return configurer
	case "16.04":
		configurer := &XenialConfigurer{
			Configurer: Configurer{
				LinuxConfigurer: configurer.LinuxConfigurer{
					Host: h,
				},
			},
		}
		return configurer
	}

	return nil
}

func init() {
	resolver.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
