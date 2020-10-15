package centos

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
)

// Configurer is the CentOS specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

// InstallBasePackages install all the needed base packages on the host
func (c *Configurer) InstallBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo yum install -y curl socat iptables iputils gzip")
}

func resolveCentosConfigurer(h *api.Host) api.HostConfigurer {
	if h.Metadata.Os.ID == "centos" {
		return &Configurer{
			Configurer: enterpriselinux.Configurer{
				LinuxConfigurer: configurer.LinuxConfigurer{
					Host: h,
				},
			},
		}
	}

	return nil
}

func init() {
	api.RegisterHostConfigurer(resolveCentosConfigurer)
}
