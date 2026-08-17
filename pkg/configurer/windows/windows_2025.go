package windows

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// Windows2025Configurer is a Windows 2025 configurer implementation.
type Windows2025Configurer struct {
	configurer.WindowsConfigurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "windows" && r.Version == "10.0.26100"
		},
		func() any {
			return Windows2025Configurer{}
		},
	)
}
