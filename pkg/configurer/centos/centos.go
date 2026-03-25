package centos

import (
	"github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
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
