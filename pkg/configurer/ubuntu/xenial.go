package ubuntu

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// XenialConfigurer is the Ubuntu Xenial specific host configurer implementation
type XenialConfigurer struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "ubuntu" && os.Version == "16.04"
		},
		func() interface{} {
			return XenialConfigurer{}
		},
	)
}
