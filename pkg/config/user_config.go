package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

const (
	configFile = "~/.mirantis-mcc/user.yaml"
)

// UserConfig struct for MCC config
type UserConfig struct {
	Name    string `yaml:"name"`
	Company string `yaml:"company"`
	Email   string `yaml:"email"`
}

// GetUserConfig returns a new decoded Config struct
func GetUserConfig() (*UserConfig, error) {
	configFile, err := homedir.Expand(configFile)
	if err != nil {
		return nil, err
	}

	config := &UserConfig{}
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

// SaveUserConfig saves config struct to yaml file
func SaveUserConfig(config *UserConfig) error {
	configFile, err := homedir.Expand(configFile)
	if err != nil {
		return err
	}
	configDir := filepath.Dir(configFile)
	if err = ensureDir(configDir); err != nil {
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
