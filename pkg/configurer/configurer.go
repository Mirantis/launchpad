package configurer

import "github.com/Mirantis/mcc/pkg/host"

type HostConfigurer interface {
	InstallEngine() error
}

func ForHost(host *host.Host) (HostConfigurer, error) {
	// TODO Get correct OS based configurer, now just default to UbuntuConfigurer
	return NewUbuntuConfigurer(host), nil
}
