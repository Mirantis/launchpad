package oracle

import (
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// Configurer is the Oracle Linux  specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "ol"
		},
		func() interface{} {
			return Configurer{}
		},
	)
}
