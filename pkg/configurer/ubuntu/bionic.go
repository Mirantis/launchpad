package ubuntu

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// BionicConfigurer is the Ubuntu Bionix specific host configurer implementation.
type BionicConfigurer struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ubuntu" && r.Version == "18.04"
		},
		func() interface{} {
			return BionicConfigurer{}
		},
	)
}
