package user

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/k0sproject/rig/util"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

const (
	configFile = "~/.mirantis-launchpad/user.yaml"
)

// Config struct for launchpad user config
type Config struct {
	Name    string `yaml:"name"`
	Company string `yaml:"company"`
	Email   string `yaml:"email"`
	Eula    bool   `yaml:"eula"`
}

// GetConfig returns a new decoded Config struct
func GetConfig() (*Config, error) {
	configFile, err := homedir.Expand(configFile)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	// Open config file
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves config struct to yaml file
func SaveConfig(config *Config) error {
	configFile, err := homedir.Expand(configFile)
	if err != nil {
		return err
	}
	configDir := filepath.Dir(configFile)
	if err = util.EnsureDir(configDir); err != nil {
		return err
	}
	d, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configFile, d, 0644)
	if err != nil {
		return err
	}
	return nil
}
