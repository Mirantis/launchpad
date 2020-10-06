package api

import "fmt"

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under api because it has direct deps to api structs
type HostConfigurer interface {
	CheckPrivilege() error
	ResolveHostname() string
	ResolveLongHostname() string
	ResolvePrivateInterface() (string, error)
	ResolveInternalIP() (string, error)
	IsContainerized() bool
	SELinuxEnabled() bool
	InstallBasePackages() error
	UpdateEnvironment() error
	CleanupEnvironment() error
	InstallEngine(engineConfig *EngineConfig) error
	UninstallEngine(engineConfig *EngineConfig) error
	DockerCommandf(template string, args ...interface{}) string
	RestartEngine() error
	ValidateFacts() error
	AuthenticateDocker(user, pass, repo string) error
	WriteFile(path, content, permissions string) error
	ReadFile(path string) (string, error)
	DeleteFile(path string) error
	FileExist(path string) bool
	HTTPStatus(url string) (int, error)
	Pwd() string
}

// HostConfigurerBuilder defines the builder function signature
type HostConfigurerBuilder func(h *Host) HostConfigurer

var hostConfigurers []HostConfigurerBuilder

// RegisterHostConfigurer registers a known host OS specific configurer builder
func RegisterHostConfigurer(adapter HostConfigurerBuilder) {
	hostConfigurers = append(hostConfigurers, adapter)
}

// GetHostConfigurers gives out all the registered configurer builders
func GetHostConfigurers() []HostConfigurerBuilder {
	return hostConfigurers
}

// ResolveHostConfigurer resolves a configurer for a host
func ResolveHostConfigurer(h *Host) error {
	configurers := GetHostConfigurers()
	for _, resolver := range configurers {
		configurer := resolver(h)
		if configurer != nil {
			h.Configurer = configurer
		}
	}
	if h.Configurer == nil {
		return fmt.Errorf("%s: has unsupported OS (%s)", h.Address, h.Metadata.Os.Name)
	}
	return nil
}
