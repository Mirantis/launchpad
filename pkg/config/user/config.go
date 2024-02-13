package user

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mirantis/mcc/pkg/util/fileutil"
	"gopkg.in/yaml.v2"
)

const (
	configFile = "~/.mirantis-launchpad/user.yaml"
)

// Config struct for launchpad user config.
type Config struct {
	Name    string `yaml:"name"`
	Company string `yaml:"company"`
	Email   string `yaml:"email"`
	Eula    bool   `yaml:"eula"`
}

// GetConfig returns a new decoded Config struct.
func GetConfig() (*Config, error) {
	configFile, err := fileutil.ExpandHomeDir(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to expand config file path: %w", err)
	}

	config := &Config{}
	// Open config file
	file, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves config struct to yaml file.
func SaveConfig(config *Config) error {
	configFile, err := fileutil.ExpandHomeDir(configFile)
	if err != nil {
		return fmt.Errorf("failed to expand config file path: %w", err)
	}
	configDir := filepath.Dir(configFile)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	d, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	err = os.WriteFile(configFile, d, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
