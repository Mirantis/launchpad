package ubuntu

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// NobleConfigurer is the Ubuntu Noble Numbat (24.04) specific host configurer implementation.
type NobleConfigurer struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ubuntu" && r.Version == "24.04"
		},
		func() interface{} {
			return NobleConfigurer{}
		},
	)
}
