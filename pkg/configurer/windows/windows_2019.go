package windows

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/configurer"
)

// Windows2019Configurer is a Windows 2019 configurer implementation.
type Windows2019Configurer struct {
	configurer.WindowsConfigurer
}

// InstallBasePackages installs the needed base packages on Ubuntu
func (c *Windows2019Configurer) InstallBasePackages() error {
	return nil
}

func resolveWindowsConfigurer(h *api.Host) api.HostConfigurer {
	if h.Metadata.Os.ID != "windows-10.0.17763" {
		return nil
	}
	switch h.Metadata.Os.Version {
	case "10.0.17763":
		configurer := &Windows2019Configurer{
			WindowsConfigurer: configurer.WindowsConfigurer{
				Host: h,
			},
		}
		return configurer
	}

	return nil
}

func init() {
	api.RegisterHostConfigurer(resolveWindowsConfigurer)
}
