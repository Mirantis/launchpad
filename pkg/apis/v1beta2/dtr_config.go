package v1beta3

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/constant"
)

// DtrConfig has all the bits needed to configure DTR during installation
type DtrConfig struct {
	Version         string   `yaml:"version"`
	ImageRepo       string   `yaml:"imageRepo"`
	InstallFlags    []string `yaml:"installFlags,flow"`
	ReplicaConfig   string   `yaml:"replicaConfig,omitempty"  default:"random"`
	LicenseFilePath string   `yaml:"licenseFilePath,omitempty" validate:"omitempty,file"`

	Metadata *DtrMetadata
}

// DtrMetadata is metadata needed by DTR for configuration and is gathered at
// the GatherFacts phase and at the end of each configuration phase
type DtrMetadata struct {
	Installed          bool
	InstalledVersion   string
	DtrLeaderAddress   string
	DtrLeaderReplicaID string
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *DtrConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawDtrConfig DtrConfig
	config := NewDtrConfig()
	raw := rawDtrConfig(config)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*c = DtrConfig(raw)
	return nil
}

// NewDtrConfig creates new config with sane defaults
func NewDtrConfig() DtrConfig {
	return DtrConfig{
		Version:   constant.DTRVersion,
		ImageRepo: constant.ImageRepo,
	}
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config
func (c *DtrConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/dtr:%s", c.ImageRepo, c.Version)
}
