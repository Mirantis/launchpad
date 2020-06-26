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
	type rawConfig EngineConfig
	config := NewEngineConfig()
	raw := rawConfig(config)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = EngineConfig(raw)
	return nil
}

// NewEngineConfig creates new default engine config struct
func NewEngineConfig() EngineConfig {
	return EngineConfig{
		Version:           constant.EngineVersion,
		Channel:           constant.EngineChannel,
		RepoURL:           constant.EngineRepoURL,
		InstallURLLinux:   constant.EngineInstallURLLinux,
		InstallURLWindows: constant.EngineInstallURLWindows,
	}
}
