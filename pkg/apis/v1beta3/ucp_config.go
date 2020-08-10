package v1beta3

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"

	"github.com/hashicorp/go-version"
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

	if raw.Cloud != nil && raw.Cloud.ConfigFile != "" {
		cloudConfigData, err := util.LoadExternalFile(raw.Cloud.ConfigFile)
		if err != nil {
			return err
		}
		raw.Cloud.ConfigData = string(cloudConfigData)
	}

	v, err := version.NewVersion(raw.Version)
	if err != nil {
		return err
	}

	if c.ImageRepo == constant.ImageRepo && c.UseNewImageRepo(v) {
		c.ImageRepo = constant.ImageRepoNew
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

func (c *UcpConfig) UseNewImageRepo(v *version.Version) bool {
	// >=3.1.15 || >=3.2.8 || >=3.3.2 is "mirantis"

	c1, _ := version.NewConstraint("< 3.2, >= 3.1.15")
	c2, _ := version.NewConstraint("> 3.1, < 3.3, >= 3.2.8")
	c3, _ := version.NewConstraint("> 3.3, < 3.4, >= 3.3.2")
	c4, _ := version.NewConstraint(">= 3.4")
	return c1.Check(v) || c2.Check(v) || c3.Check(v) || c4.Check(v)
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config
func (c *UcpConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/ucp:%s", c.ImageRepo, c.Version)
}
