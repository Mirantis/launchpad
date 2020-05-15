package centos

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/configurer"
)

// Configurer is the CentOS specific implementation of a host configurer
type Configurer struct {
	configurer.LinuxConfigurer
}

// InstallBasePackages install all the needed base packages on the host
func (c *Configurer) InstallBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo yum install -y curl")
}

func resolveCentosConfigurer(h *config.Host) config.HostConfigurer {
	if h.Metadata.Os.ID == "centos" {
		return &Configurer{
			LinuxConfigurer: configurer.LinuxConfigurer{
				Host: h,
			},
		}
	}

	return nil
}

func init() {
	config.RegisterHostConfigurer(resolveCentosConfigurer)
}
