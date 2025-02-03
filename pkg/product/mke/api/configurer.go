package api

import (
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/k0sproject/rig/os"
)

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under api because it has direct deps to api structs.
type HostConfigurer interface {
	CheckPrivilege(os.Host) error
	Hostname(os.Host) string
	LongHostname(os.Host) string
	ResolvePrivateInterface(os.Host) (string, error)
	ResolveInternalIP(os.Host, string, string) (string, error)
	IsContainer(os.Host) bool
	FixContainer(os.Host) error
	SELinuxEnabled(os.Host) bool
	InstallMKEBasePackages(os.Host) error
	UpdateEnvironment(os.Host, map[string]string) error
	CleanupEnvironment(os.Host, map[string]string) error
	MCRConfigPath() string
	InstallMCR(os.Host, string, common.MCRConfig) error
	UninstallMCR(os.Host, string, common.MCRConfig) error
	DockerCommandf(template string, args ...interface{}) string
	RestartMCR(os.Host) error
	AuthenticateDocker(h os.Host, user, pass, repo string) error
	LocalAddresses(os.Host) ([]string, error)
	ValidateLocalhost(os.Host) error
	WriteFile(os.Host, string, string, string) error
	ReadFile(os.Host, string) (string, error)
	DeleteFile(os.Host, string) error
	FileExist(os.Host, string) bool
	HTTPStatus(os.Host, string) (int, error)
	Pwd(os.Host) string
	JoinPath(...string) string
	Reboot(os.Host) error
	AuthorizeDocker(os.Host) error
}
