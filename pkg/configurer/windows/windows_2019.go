package windows

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/configurer/resolver"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// Windows2019Configurer is a Windows 2019 configurer implementation.
type Windows2019Configurer struct {
	configurer.WindowsConfigurer
}

// InstallMKEBasePackages installs the needed base packages on Ubuntu
func (c *Windows2019Configurer) InstallMKEBasePackages() error {
	return nil
}

func resolveWindowsConfigurer(h configurer.Host, os *common.OsRelease) interface{} {
	if os.ID != "windows-10.0.17763" {
		return nil
	}
	switch os.Version {
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
	resolver.RegisterHostConfigurer(resolveWindowsConfigurer)
}
