package v1beta1

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under v1beta1 because it has direct deps to api structs
type HostConfigurer interface {
	ResolveHostname() string
	ResolveInternalIP() string
	IsContainerized() bool
	InstallBasePackages() error
	InstallEngine(engineConfig *EngineConfig) error
	UninstallEngine(engineConfig *EngineConfig) error
	DockerCommandf(template string, args ...interface{}) string
	RestartEngine() error
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
