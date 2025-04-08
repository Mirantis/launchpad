package ubuntu

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// NobleConfigurer is the Ubuntu Noble Numbat (24.04) specific host configurer implementation.
type NobleConfigurer struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "ubuntu" && os.Version == "24.04"
		},
		func() interface{} {
			return NobleConfigurer{}
		},
	)
}
