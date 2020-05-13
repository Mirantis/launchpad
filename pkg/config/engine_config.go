package config

type EngineConfig struct {
	Version    string
	RepoURL    string
	InstallURL string
	Channel    string
}

func (c *EngineConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig EngineConfig
	raw := NewEngineConfig()

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
