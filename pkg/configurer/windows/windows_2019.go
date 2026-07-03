package windows

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	rigos "github.com/k0sproject/rig/v2/os"
)

// Windows2019Configurer is a Windows 2019 configurer implementation.
type Windows2019Configurer struct {
	configurer.WindowsConfigurer
}

func init() {
	configurer.RegisterOSModule(
		func(r *rigos.Release) bool {
			return r.ID == "windows" && r.Version == "10.0.17763"
		},
		func() any {
			return Windows2019Configurer{}
		},
	)
}
