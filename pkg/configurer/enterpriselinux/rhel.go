package enterpriselinux

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// Rhel RedHat Enterprise Linux.
type Rhel struct {
	Configurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "rhel"
		},
		func() any {
			return Rhel{}
		},
	)
}
