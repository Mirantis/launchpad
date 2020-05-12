package host

type HostConfigurer interface {
	ResolveHostname() string
	ResolveInternalIP() string
	InstallBasePackages() error
	InstallEngine() error
}

type HostConfigurerBuilder func(h *Host) HostConfigurer

var hostConfigurers []HostConfigurerBuilder

func RegisterHostConfigurer(adapter HostConfigurerBuilder) {
	hostConfigurers = append(hostConfigurers, adapter)
}

func GetHostConfigurers() []HostConfigurerBuilder {
	return hostConfigurers
}
