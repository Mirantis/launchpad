package centos

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/mcc/pkg/configurer/resolver"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// Configurer is the CentOS specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

// InstallMKEBasePackages install all the needed base packages on the host
func (c *Configurer) InstallMKEBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo yum install -y curl socat iptables iputils gzip")
}

// InstallK0sBasePackages install all the needed base packages on the host
func (c *Configurer) InstallK0sBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo yum install -y curl gzip")
}

func resolveCentosConfigurer(h configurer.Host, os *common.OsRelease) interface{} {
	if os.ID == "centos" {
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
	resolver.RegisterHostConfigurer(resolveCentosConfigurer)
}
