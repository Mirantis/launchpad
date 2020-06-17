package ubuntu

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/configurer"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	configurer.LinuxConfigurer
}

// InstallBasePackages installs the needed base packages on Ubuntu
func (c *Configurer) InstallBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q curl apt-utils socat")
}

// UninstallEngine uninstalls docker-ee engine
func (c *Configurer) UninstallEngine(engineConfig *api.EngineConfig) error {
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

func resolveUbuntuConfigurer(h *api.Host) api.HostConfigurer {
	if h.Metadata.Os.ID != "ubuntu" {
		return nil
	}
	switch h.Metadata.Os.Version {
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
	api.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
