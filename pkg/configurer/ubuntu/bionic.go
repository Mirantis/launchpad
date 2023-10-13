package ubuntu

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// BionicConfigurer is the Ubuntu Bionix specific host configurer implementation.
type BionicConfigurer struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "ubuntu" && os.Version == "18.04"
		},
		func() interface{} {
			return BionicConfigurer{}
		},
	)
}
