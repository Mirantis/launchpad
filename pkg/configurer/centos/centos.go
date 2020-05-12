package centos

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/host"
)

type CentOSConfigurer struct {
	configurer.LinuxConfigurer
}

func (c *CentOSConfigurer) InstallBasePackages() error {
	return c.Host.Exec("sudo yum install -y curl")
}

func resolveCentosConfigurer(h *host.Host) host.HostConfigurer {
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
	host.RegisterHostConfigurer(resolveCentosConfigurer)
}
