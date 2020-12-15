package oracle

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

// Configurer is the CentOS specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

func resolveOracleConfigurer(h *api.Host) api.HostConfigurer {
	if h.Metadata.Os.ID == "ol" {
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
	api.RegisterHostConfigurer(resolveOracleConfigurer)
}
