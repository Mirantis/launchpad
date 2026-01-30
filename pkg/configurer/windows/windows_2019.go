package windows

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// Windows2019Configurer is a Windows 2019 configurer implementation.
type Windows2019Configurer struct {
	configurer.WindowsConfigurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "windows" && os.Version == "10.0.17763"
		},
		func() any {
			return Windows2019Configurer{}
		},
	)
}
