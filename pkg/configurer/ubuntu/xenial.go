package ubuntu

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// XenialConfigurer is the Ubuntu Xenial specific host configurer implementation.
type XenialConfigurer struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "ubuntu" && r.Version == "16.04"
		},
		func() interface{} {
			return XenialConfigurer{}
		},
	)
}
