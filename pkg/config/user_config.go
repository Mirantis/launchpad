package config

import (
	"io/ioutil"
	"os"

	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

const (
	configPath = "~/.mirantis-mcc"
)

// UserConfig struct for MCC config
type UserConfig struct {
	Name    string `yaml:"name"`
	Company string `yaml:"company"`
	Email   string `yaml:"email"`
}

// GetUserConfig returns a new decoded Config struct
func GetUserConfig() (*UserConfig, error) {
	configPath, err := homedir.Expand(configPath)
	if err != nil {
		return nil, err
	}

	config := &UserConfig{}
	// Open config file
	file, err := os.Open(configPath + "/user.yaml")
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
	configPath, err := homedir.Expand(configPath)
	ensureConfigDir()
	d, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configPath+"/user.yaml", d, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ensureConfigDir() error {
	configPath, err := homedir.Expand(configPath)
	if err != nil {
		return err
	}

	if _, serr := os.Stat(configPath); os.IsNotExist(serr) {
		merr := os.MkdirAll(configPath, os.ModePerm)
		if merr != nil {
			return err
		}
	}
	return nil
}
