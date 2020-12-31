package resolver

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/configurer"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

// HostConfigurerBuilder defines the builder function signature
type HostConfigurerBuilder func(h configurer.Host, os *common.OsRelease) interface{}

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
func ResolveHostConfigurer(h configurer.Host, os *common.OsRelease) (interface{}, error) {
	configurers := GetHostConfigurers()
	for _, resolver := range configurers {
		if configurer := resolver(h, os); configurer != nil {
			return configurer, nil
		}
	}

	return nil, fmt.Errorf("%s: has unsupported OS (%s)", h, os)
}
