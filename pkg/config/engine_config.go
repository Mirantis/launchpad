package config

type EngineConfig struct {
	Version    string `yaml:"version"`
	RepoURL    string `yaml:"repoUrl"`
	InstallURL string `yaml:"installURL"`
	Channel    string `yaml:"channel"`
}

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

func NewEngineConfig() EngineConfig {
	return EngineConfig{
		Version:    EngineVersion,
		Channel:    EngineChannel,
		RepoURL:    EngineRepoURL,
		InstallURL: EngineInstallURL,
	}
}
