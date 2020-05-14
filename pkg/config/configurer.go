package config

type HostConfigurer interface {
	ResolveHostname() string
	ResolveInternalIP() string
	IsContainerized() bool
	InstallBasePackages() error
	InstallEngine(engineConfig *EngineConfig) error
}

type HostConfigurerBuilder func(h *Host) HostConfigurer

var hostConfigurers []HostConfigurerBuilder

func RegisterHostConfigurer(adapter HostConfigurerBuilder) {
	hostConfigurers = append(hostConfigurers, adapter)
}

func GetHostConfigurers() []HostConfigurerBuilder {
	return hostConfigurers
}
