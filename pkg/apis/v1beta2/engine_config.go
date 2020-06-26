package v1beta2

import (
	"github.com/Mirantis/mcc/pkg/constant"
)

// EngineConfig holds the engine installation specific options
type EngineConfig struct {
	Version           string `yaml:"version"`
	RepoURL           string `yaml:"repoUrl,omitempty"`
	InstallURLLinux   string `yaml:"installURLLinux,omitempty"`
	InstallURLWindows string `yaml:"installURLWindows,omitempty"`
	Channel           string `yaml:"channel,omitempty"`
}

// UnmarshalYAML puts in sane defaults when unmarshaling from yaml
func (c *EngineConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.SetDefaults()

	type yEngineConfig EngineConfig
	yc := (*yEngineConfig)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	return nil
}

// SetDefaults sets defaults on the object
func (c *EngineConfig) SetDefaults() {
	c.Version = constant.EngineVersion
	c.Channel = constant.EngineChannel
	c.RepoURL = constant.EngineRepoURL
	c.InstallURLLinux = constant.EngineInstallURLLinux
	c.InstallURLWindows = constant.EngineInstallURLWindows
}
