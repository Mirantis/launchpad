package api

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/configurer/resolver"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/creasty/defaults"
)

// Host contains all the needed details to work with hosts
type Host struct {
	Role         string            `yaml:"role" validate:"oneof=server worker"`
	UploadBinary bool              `yaml:"uploadBinary,omitempty"`
	K0sBinary    string            `yaml:"k0sBinary,omitempty" validate:"omitempty,file"`
	Environment  map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks        common.Hooks      `yaml:"hooks,omitempty" validate:"dive,keys,oneof=apply reset,endkeys,dive,keys,oneof=before after,endkeys,omitempty"`

	common.ConnectableHost `yaml:",inline"`

	InitSystem common.InitSystem `yaml:"-"`
	Configurer HostConfigurer    `yaml:"-"`
	Metadata   *HostMetadata     `yaml:"-"`

	name string
}

// HostMetadata resolved metadata for host
type HostMetadata struct {
	K0sVersion string
	Arch       string
	Os         *common.OsRelease
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(h)
	defaults.Set(h.ConnectableHost)

	type host Host
	yh := (*host)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	if h.WinRM == nil && h.SSH == nil && !h.Localhost {
		h.SSH = common.DefaultSSH()
	}

	return nil
}

// ResolveHostConfigurer will resolve and cast a configurer for the K0s configurer interface
func (h *Host) ResolveHostConfigurer() error {
	if h.Metadata == nil || h.Metadata.Os == nil {
		return fmt.Errorf("%s: OS not known", h)
	}
	r, err := resolver.ResolveHostConfigurer(h, h.Metadata.Os)
	if err != nil {
		return err
	}

	if configurer, ok := r.(HostConfigurer); ok {
		h.Configurer = configurer
		return nil
	}

	return fmt.Errorf("%s: has unsupported OS (%s)", h, h.Metadata.Os)
}
