package v1beta1

import (
	"encoding/json"

	"github.com/Mirantis/mcc/pkg/constant"
)

// EngineConfig holds the engine installation specific options
type EngineConfig struct {
	Version        string             `yaml:"version"`
	RepoURL        string             `yaml:"repoUrl"`
	InstallURL     string             `yaml:"installURL"`
	Channel        string             `yaml:"channel"`
	Configurations []EngineHostConfig `yaml:"config"`
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
		Version:    constant.EngineVersion,
		Channel:    constant.EngineChannel,
		RepoURL:    constant.EngineRepoURL,
		InstallURL: constant.EngineInstallURL,
	}
}

// GetDaemonConfig gets the named daemon config
func (c *EngineConfig) GetDaemonConfig(name string) *EngineHostConfig {
	for _, ehc := range c.Configurations {
		if name == ehc.Name {
			return &ehc
		}
	}

	return nil
}

// EngineHostConfig defines the named host level engine config options, essentially the daemon.json equivalent which will
// be injected into hosts when installing engine.
type EngineHostConfig struct {
	Name   string
	Config map[string]interface{}
}

// ToDaemonJSON converts the engine config into daemon.json data
func (ec *EngineHostConfig) ToDaemonJSON() ([]byte, error) {
	return json.Marshal(ec.Config)
}
