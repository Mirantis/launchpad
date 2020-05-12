package ubuntu

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/host"
)

type UbuntuConfigurer struct {
	configurer.LinuxConfigurer
}

func (c *UbuntuConfigurer) InstallBasePackages() error {
	return c.Host.Exec("sudo apt-get update && sudo apt-get install -y curl apt-utils")
}

func resolveUbuntuConfigurer(h *host.Host) host.HostConfigurer {
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
	host.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
