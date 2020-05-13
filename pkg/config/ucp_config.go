package config

type UcpConfig struct {
	Version      string   `yaml:"version"`
	ImageRepo    string   `yaml:"imageRepo"`
	InstallFlags []string `yaml:"installFlags,flow"`
}

func (c *UcpConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawUcpConfig UcpConfig
	config := NewUcpConfig()
	raw := rawUcpConfig(config)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = UcpConfig(raw)
	return nil
}

func NewUcpConfig() UcpConfig {
	return UcpConfig{
		Version:   Version,
		ImageRepo: ImageRepo,
	}
}
