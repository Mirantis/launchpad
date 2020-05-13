package centos

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/configurer"
)

type CentOSConfigurer struct {
	configurer.LinuxConfigurer
}

func (c *CentOSConfigurer) InstallBasePackages() error {
	return c.Host.Exec("sudo yum install -y curl")
}

func resolveCentosConfigurer(h *config.Host) config.HostConfigurer {
	if h.Metadata.Os.ID == "centos" {
		return &CentOSConfigurer{
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
