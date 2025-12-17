package enterpriselinux

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// Rhel RedHat Enterprise Linux.
type Rhel struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "rhel"
		},
		func() any {
			return Rhel{}
		},
	)
}
