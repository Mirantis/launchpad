package oracle

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	"github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
	rigos "github.com/k0sproject/rig/v2/os"
)

// Configurer is the Oracle Linux  specific implementation of a host configurer.
type Configurer struct {
	enterpriselinux.Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ol"
		},
		func() any {
			return Configurer{}
		},
	)
}
