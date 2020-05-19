package centos

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
)

// Configurer is the CentOS specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
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
