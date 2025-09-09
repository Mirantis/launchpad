package configurer

import "errors"

var (
	// ErrorConfigurerMCRInstall MCR installation in the configurer failed.
	ErrorConfigurerMCRInstall = errors.New("MCR Installation failed in the configurer")
	// ErrorConfigurerMCRUninstall MCR uninstallation in the configurer failed.
	ErrorConfigurerMCRUninstall = errors.New("MCR Uninstallatioon failed in the configurer")
)
