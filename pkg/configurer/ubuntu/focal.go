package ubuntu

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// FocalConfigurer is the Ubuntu Focal (20.04) specific host configurer implementation.
type FocalConfigurer struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ubuntu" && r.Version == "20.04"
		},
		func() interface{} {
			return FocalConfigurer{}
		},
	)
}
