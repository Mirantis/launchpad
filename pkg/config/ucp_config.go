package config

import (
	"fmt"
	"io/ioutil"

	"github.com/mitchellh/go-homedir"
)

type UcpConfig struct {
	Version      string   `yaml:"version"`
	ImageRepo    string   `yaml:"imageRepo"`
	InstallFlags []string `yaml:"installFlags,flow"`
	ConfigFile   string   `yaml:"configFile" validate:"file"`
	ConfigData   string   `yaml:"configData"`

	Metadata *UcpMetadata
}

type UcpMetadata struct {
	Installed        bool
	InstalledVersion string
}

func (c *UcpConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawUcpConfig UcpConfig
	config := NewUcpConfig()
	raw := rawUcpConfig(config)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if raw.ConfigFile != "" {
		configfile, err := homedir.Expand(raw.ConfigFile)
		if err != nil {
			return err
		}

		cfg, err := ioutil.ReadFile(configfile)
		if err != nil {
			return err
		}
		raw.ConfigData = string(cfg)
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

func (u *UcpConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/ucp:%s", u.ImageRepo, u.Version)
}
