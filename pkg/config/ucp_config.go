package config

type UcpConfig struct {
	Version   string
	ImageRepo string
}

func (c *UcpConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig UcpConfig
	raw := NewUcpConfig()

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
