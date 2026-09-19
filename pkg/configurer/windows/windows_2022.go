package windows

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// Windows2022Configurer is a Windows 2022 configurer implementation.
type Windows2022Configurer struct {
	configurer.WindowsConfigurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "windows" && r.Version == "10.0.20348"
		},
		func() any {
			return Windows2022Configurer{}
		},
	)
}
