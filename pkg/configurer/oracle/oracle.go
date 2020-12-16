package oracle

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// Configurer is the Oracle Linux  specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

func resolveOracleConfigurer(h *api.Host) api.HostConfigurer {
	if h.Metadata.Os.ID == "ol" {
		log.Warnf("%s: Oracle Linux support is still at beta stage and under development", h)
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
