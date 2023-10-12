package ubuntu

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// JammyConfigurer is the Ubuntu Focal (20.04) specific host configurer implementation.
type JammyConfigurer struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "ubuntu" && os.Version == "22.04"
		},
		func() interface{} {
			return JammyConfigurer{}
		},
	)
}
