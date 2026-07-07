package ubuntu

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// JammyConfigurer is the Ubuntu Jammy Jellyfish (22.04) specific host configurer implementation.
type JammyConfigurer struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ubuntu" && r.Version == "22.04"
		},
		func() interface{} {
			return JammyConfigurer{}
		},
	)
}
