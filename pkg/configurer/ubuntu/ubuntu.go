package ubuntu

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/configurer"
)

type UbuntuConfigurer struct {
	configurer.LinuxConfigurer
}

func (c *UbuntuConfigurer) InstallBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo apt-get update && sudo apt-get install -y curl apt-utils")
}

func resolveUbuntuConfigurer(h *config.Host) config.HostConfigurer {
	if h.Metadata.Os.ID != "ubuntu" {
		return nil
	}
	switch h.Metadata.Os.Version {
	case "18.04":
		configurer := &BionicConfigurer{
			UbuntuConfigurer: UbuntuConfigurer{
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
	config.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
