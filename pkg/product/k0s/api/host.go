package api

import (
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/creasty/defaults"
	"gopkg.in/yaml.v2"
)

// Host contains all the needed details to work with hosts
type Host struct {
	Role         string            `yaml:"role" validate:"oneof=server worker"`
	UploadBinary bool              `yaml:"uploadBinary,omitempty"`
	K0sBinary    string            `yaml:"k0sBinary,omitempty" validate:"omitempty,file"`
	Configurer   HostConfigurer    `yaml:"-"`
	Metadata     *HostMetadata     `yaml:"-"`
	Environment  map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks        common.Hooks      `yaml:"hooks,omitempty" validate:"dive,keys,oneof=apply reset,endkeys,dive,keys,oneof=before after,endkeys,omitempty"`

	common.ConnectableHost `yaml:",inline"`

	name string
}

// HostMetadata resolved metadata for host
type HostMetadata struct {
	Hostname        string
	LongHostname    string
	InternalAddress string
	K0sVersion      string
	Os              *common.OsRelease
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

// K0sVersion returns installed version of k0s
// TODO: need to be in configurer
func (h *Host) K0sVersion() (string, error) {
	return h.ExecWithOutput("k0s version")
}

// ConfigureK0s persists k0s.yaml config to the host
func (h *Host) ConfigureK0s(config *common.GenericHash) error {
	output, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return h.Configurer.WriteFile(h.Configurer.K0sConfigPath(), string(output), "0700")
}
