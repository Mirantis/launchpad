package v1beta2

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
)

// UcpConfig has all the bits needed to configure UCP during installation
type UcpConfig struct {
	Version         string    `yaml:"version"`
	ImageRepo       string    `yaml:"imageRepo,omitempty"`
	InstallFlags    []string  `yaml:"installFlags,omitempty,flow"`
	ConfigFile      string    `yaml:"configFile,omitempty" validate:"omitempty,file"`
	ConfigData      string    `yaml:"configData,omitempty"`
	LicenseFilePath string    `yaml:"licenseFilePath,omitempty" validate:"omitempty,file"`
	Cloud           *UcpCloud `yaml:"cloud,omitempty"`

	Metadata *UcpMetadata `yaml:"-"`
}

// UcpMetadata has the "runtime" discovered metadata of already existing installation.
type UcpMetadata struct {
	Installed        bool
	InstalledVersion string
	ClusterID        string
}

// UcpCloud has the cloud provider configuration
type UcpCloud struct {
	Provider   string `yaml:"provider,omitempty" validate:"required"`
	ConfigFile string `yaml:"configFile,omitempty" validate:"omitempty,file"`
	ConfigData string `yaml:"configData,omitempty"`
}

func (c *UcpConfig) getInstallFlagValue(name string) string {
	for _, flag := range c.InstallFlags {
		if strings.HasPrefix(flag, fmt.Sprintf("%s=", name)) {
			values := strings.SplitN(flag, "=", 2)
			if values[1] != "" {
				return values[1]
			}
		}
		if strings.HasPrefix(flag, fmt.Sprintf("%s ", name)) {
			values := strings.SplitN(flag, " ", 2)
			if values[1] != "" {
				return values[1]
			}
		}
	}
	return ""
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *UcpConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawUcpConfig UcpConfig
	config := NewUcpConfig()
	raw := rawUcpConfig(config)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if raw.ConfigFile != "" {
		configData, err := util.LoadExternalFile(raw.ConfigFile)
		if err != nil {
			return err
		}
		raw.ConfigData = string(configData)
	}

	*c = UcpConfig(raw)
	return nil
}

// NewUcpConfig creates new config with sane defaults
func NewUcpConfig() UcpConfig {
	return UcpConfig{
		Version:   constant.UCPVersion,
		ImageRepo: constant.ImageRepo,
	}
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config
func (c *UcpConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/ucp:%s", c.ImageRepo, c.Version)
}

// IsCustomImageRepo checks if the config is using a custom image repo
func (c *UcpConfig) IsCustomImageRepo() bool {
	return c.ImageRepo != constant.ImageRepo
}
