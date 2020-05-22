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
	return c.Host.Exec("sudo apt-get update && sudo apt-get install -y curl apt-utils")
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
	}

	return nil
}

func init() {
	api.RegisterHostConfigurer(resolveUbuntuConfigurer)
}
