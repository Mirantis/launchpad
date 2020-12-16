package enterpriselinux

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/resolver"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// Rhel RedHat Enterprise Linux
type Rhel struct {
	Configurer
}

func resolveRedhatConfigurer(h configurer.Host, os *common.OsRelease) interface{} {
	if os.ID == "rhel" {
		return &Rhel{
			Configurer: Configurer{
				LinuxConfigurer: configurer.LinuxConfigurer{
					Host: h,
				},
			},
		}
	}

	return nil
}

func init() {
	resolver.RegisterHostConfigurer(resolveRedhatConfigurer)
}
