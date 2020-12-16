package oracle

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/Mirantis/mcc/pkg/configurer/resolver"
	common "github.com/Mirantis/mcc/pkg/product/common/api"

	log "github.com/sirupsen/logrus"
)

// Configurer is the Oracle Linux  specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

func resolveOracleConfigurer(h configurer.Host, os *common.OsRelease) interface{} {
	if os.ID == "ol" {
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
	resolver.RegisterHostConfigurer(resolveOracleConfigurer)
}
