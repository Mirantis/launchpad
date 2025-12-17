package centos

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"

	"github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
)

// Configurer is the CentOS specific implementation of a host configurer.
type Configurer struct {
	enterpriselinux.Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "centos"
		},
		func() any {
			return Configurer{}
		},
	)
}
