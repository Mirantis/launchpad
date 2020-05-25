package v1beta1

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
)

// UcpConfig has all the bits needed to configure UCP during installation
type UcpConfig struct {
	Version         string   `yaml:"version"`
	ImageRepo       string   `yaml:"imageRepo"`
	InstallFlags    []string `yaml:"installFlags,flow"`
	ConfigFile      string   `yaml:"configFile" validate:"omitempty,file"`
	ConfigData      string   `yaml:"configData"`
	LicenseFilePath string   `yaml:"licenseFilePath" validate:"omitempty,file"`

	Metadata *UcpMetadata
}

// UcpMetadata has the "runtime" discovered metadata of already existing installation.
type UcpMetadata struct {
	Installed        bool
	InstalledVersion string
	ClusterID        string
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
