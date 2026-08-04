package enterpriselinux

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// RockyLinux support.
type RockyLinux struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "rocky"
		},
		func() any {
			return RockyLinux{}
		},
	)
}
