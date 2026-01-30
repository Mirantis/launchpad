package windows

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// Windows2022Configurer is a Windows 2022 configurer implementation.
type Windows2022Configurer struct {
	configurer.WindowsConfigurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "windows" && os.Version == "10.0.20348"
		},
		func() any {
			return Windows2022Configurer{}
		},
	)
}
