package api

import (
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under api because it has direct deps to api structs
type HostConfigurer interface {
	CheckPrivilege() error
	ResolveHostname() string
	ResolveLongHostname() string
	ResolvePrivateInterface() (string, error)
	ResolveInternalIP(string, string) (string, error)
	IsContainerized() bool
	SELinuxEnabled() bool
	InstallMKEBasePackages() error
	UpdateEnvironment(map[string]string) error
	CleanupEnvironment(map[string]string) error
	MCRConfigPath() string
	InstallMCR(string, common.MCRConfig) error
	UninstallMCR(string, common.MCRConfig) error
	DockerCommandf(template string, args ...interface{}) string
	RestartMCR() error
	AuthenticateDocker(user, pass, repo string) error
	LocalAddresses() ([]string, error)
	ValidateLocalhost() error
	WriteFile(path, content, permissions string) error
	ReadFile(path string) (string, error)
	DeleteFile(path string) error
	FileExist(path string) bool
	HTTPStatus(url string) (int, error)
	Pwd() string
	JoinPath(...string) string
	RebootCommand() string
}
