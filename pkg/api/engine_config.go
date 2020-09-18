package api

import (
	"github.com/Mirantis/mcc/pkg/constant"
)

// EngineConfig holds the engine installation specific options
type EngineConfig struct {
	Version           string `yaml:"version"`
	RepoURL           string `yaml:"repoURL,omitempty"`
	InstallURLLinux   string `yaml:"installURLLinux,omitempty"`
	InstallURLWindows string `yaml:"installURLWindows,omitempty"`
	Channel           string `yaml:"channel,omitempty"`
}

// UnmarshalYAML puts in sane defaults when unmarshaling from yaml
func (c *EngineConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type yEngineConfig EngineConfig
	yc := (*yEngineConfig)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	c.SetDefaults()

	return nil
}

// SetDefaults sets defaults on the object
func (c *EngineConfig) SetDefaults() {
	// Constants can't be used in tags, so yaml defaults can't be used here.
	if c.Version == "" {
		c.Version = constant.EngineVersion
	}

	if c.Channel == "" {
		c.Channel = constant.EngineChannel
	}

	if c.RepoURL == "" {
		c.RepoURL = constant.EngineRepoURL
	}

	if c.InstallURLLinux == "" {
		c.InstallURLLinux = constant.EngineInstallURLLinux
	}

	if c.InstallURLWindows == "" {
		c.InstallURLWindows = constant.EngineInstallURLWindows
	}
}
