package config

import (
	"github.com/Mirantis/launchpad/pkg/configurer"
	common "github.com/Mirantis/launchpad/pkg/product/common/config"
)

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under api because it has direct deps to api structs.
type HostConfigurer interface {
	CheckPrivilege(configurer.Host) error
	ResolvePrivateInterface(configurer.Host) (string, error)
	ResolveInternalIP(configurer.Host, string, string) (string, error)
	SELinuxEnabled(configurer.Host) bool
	UpdateEnvironment(configurer.Host, map[string]string) error
	CleanupEnvironment(configurer.Host, map[string]string) error
	MCRConfigPath() string
	InstallMCRLicense(configurer.Host, string) error
	InstallMCR(configurer.Host, common.MCRConfig) error
	UninstallMCR(configurer.Host, common.MCRConfig) error
	DockerCommandf(template string, args ...any) string
	RestartMCR(configurer.Host) error
	AuthenticateDocker(h configurer.Host, user, pass, repo string) error
	LocalAddresses(configurer.Host) ([]string, error)
	ValidateLocalhost(configurer.Host) error
	Pwd(configurer.Host) string
	Reboot(configurer.Host) error
	AuthorizeDocker(configurer.Host) error
	PrepareHost(configurer.Host) error
}
