package windows

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

// Windows2025Configurer is a Windows 2025 configurer implementation.
type Windows2025Configurer struct {
	configurer.WindowsConfigurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "windows" && os.Version == "10.0.26100"
		},
		func() any {
			return Windows2025Configurer{}
		},
	)
}
