package api

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/configurer/resolver"
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
	EngineConfigPath() string
	InstallEngine(string, common.EngineConfig) error
	UninstallEngine(string, common.EngineConfig) error
	DockerCommandf(template string, args ...interface{}) string
	RestartEngine() error
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

// ResolveHostConfigurer will resolve and cast a configurer for the MKE configurer interface
func ResolveHostConfigurer(h *Host) error {
	if h.Metadata == nil || h.Metadata.Os == nil {
		return fmt.Errorf("%s: OS not known", h)
	}
	r, err := resolver.ResolveHostConfigurer(h, h.Metadata.Os)
	if err != nil {
		return err
	}
	h.Configurer = r.(HostConfigurer)
	return nil
}
