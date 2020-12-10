package api

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/product/common/api"
	// "github.com/Mirantis/mcc/pkg/product/k0s/api"
)

// HostConfigurer defines the interface each host OS specific configurers implement.
// This is under api because it has direct deps to api structs
type HostConfigurer interface {
	CheckPrivilege() error
	ResolveHostname() string
	ResolveLongHostname() string
	SELinuxEnabled() bool
	InstallBasePackages() error
	K0sConfigPath() string
	K0sJoinToken() string
	InstallK0s(version string, k0sConfig *api.GenericHash) error
	UploadK0s(version string, k0sConfig *api.GenericHash) error
	ValidateFacts() error
	WriteFile(path, content, permissions string) error
	WriteFileLarge(content, permissions string) error
	ReadFile(path string) (string, error)
	DeleteFile(path string) error
	FileExist(path string) bool
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
		return fmt.Errorf("%s: has unsupported OS (%s)", h, h.Metadata.Os.Name)
	}
	return nil
}
