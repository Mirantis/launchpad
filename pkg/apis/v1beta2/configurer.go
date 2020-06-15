package v1beta2

import "fmt"

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under v1beta2 because it has direct deps to api structs
type HostConfigurer interface {
	ResolveHostname() string
	ResolveInternalIP() (string, error)
	IsContainerized() bool
	SELinuxEnabled() bool
	InstallBasePackages() error
	InstallEngine(engineConfig *EngineConfig) error
	UninstallEngine(engineConfig *EngineConfig) error
	DockerCommandf(template string, args ...interface{}) string
	RestartEngine() error
	ValidateFacts() error
	AuthenticateDocker(user, pass, repo string) error
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
