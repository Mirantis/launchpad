package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util/fileutil"
	"github.com/hashicorp/go-version"
)

// MKEConfig has all the bits needed to configure mke during installation.
type MKEConfig struct {
	Version          string       `yaml:"version" validate:"required"`
	ImageRepo        string       `yaml:"imageRepo,omitempty"`
	AdminUsername    string       `yaml:"adminUsername,omitempty"`
	AdminPassword    string       `yaml:"adminPassword,omitempty"`
	InstallFlags     common.Flags `yaml:"installFlags,omitempty,flow"`
	UpgradeFlags     common.Flags `yaml:"upgradeFlags,omitempty,flow"`
	ConfigFile       string       `yaml:"configFile,omitempty" validate:"omitempty,file"`
	ConfigData       string       `yaml:"configData,omitempty"`
	LicenseFilePath  string       `yaml:"licenseFilePath,omitempty" validate:"omitempty,file"`
	CACertPath       string       `yaml:"caCertPath,omitempty" validate:"omitempty,file"`
	CertPath         string       `yaml:"certPath,omitempty" validate:"omitempty,file"`
	KeyPath          string       `yaml:"keyPath,omitempty" validate:"omitempty,file"`
	CACertData       string       `yaml:"caCertData,omitempty"`
	CertData         string       `yaml:"certData,omitempty"`
	KeyData          string       `yaml:"keyData,omitempty"`
	Cloud            *MKECloud    `yaml:"cloud,omitempty"`
	NodesHealthRetry uint         `yaml:"nodesHealthRetry,omitempty" default:"0"`

	Metadata *MKEMetadata `yaml:"-"`
}

// MKEMetadata has the "runtime" discovered metadata of already existing installation.
type MKEMetadata struct {
	Installed               bool
	InstalledVersion        string
	InstalledBootstrapImage string
	ClusterID               string
	VXLAN                   bool
}

// MKECloud has the cloud provider configuration.
type MKECloud struct {
	Provider   string `yaml:"provider,omitempty" validate:"required"`
	ConfigFile string `yaml:"configFile,omitempty" validate:"omitempty,file"`
	ConfigData string `yaml:"configData,omitempty"`
}

var errMKEConfigInvalid = errors.New("invalid MKE config")

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *MKEConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type mke MKEConfig
	config := NewMKEConfig()
	raw := mke(config)
	raw.Metadata = &MKEMetadata{}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	if raw.ConfigFile != "" {
		configData, err := fileutil.LoadExternalFile(raw.ConfigFile)
		if err != nil {
			return fmt.Errorf("error in field spec.mke.configFile: %w", err)
		}
		raw.ConfigData = string(configData)
	}

	if raw.Cloud != nil && raw.Cloud.ConfigFile != "" {
		cloudConfigData, err := fileutil.LoadExternalFile(raw.Cloud.ConfigFile)
		if err != nil {
			return fmt.Errorf("error in field spec.mke.cloud.configFile: %w", err)
		}
		raw.Cloud.ConfigData = string(cloudConfigData)
	}

	if flagValue := raw.InstallFlags.GetValue("--admin-username"); flagValue != "" {
		if raw.AdminUsername == "" {
			raw.AdminUsername = flagValue
			raw.InstallFlags.Delete("--admin-username")
		} else if flagValue != raw.AdminUsername {
			return fmt.Errorf("%w: both Spec.mke.AdminUsername and Spec.mke.InstallFlags --admin-username set, only one allowed", errMKEConfigInvalid)
		}
	}

	if flagValue := raw.InstallFlags.GetValue("--admin-password"); flagValue != "" {
		if raw.AdminPassword == "" {
			raw.AdminPassword = flagValue
			raw.InstallFlags.Delete("--admin-password")
		} else if flagValue != raw.AdminPassword {
			return fmt.Errorf("%w: both Spec.mke.AdminPassword and Spec.mke.InstallFlags --admin-password set, only one allowed", errMKEConfigInvalid)
		}
	}

	if flagValue := raw.UpgradeFlags.GetValue("--admin-username"); flagValue != "" {
		if raw.AdminUsername == "" {
			raw.AdminUsername = flagValue
			raw.UpgradeFlags.Delete("--admin-username")
		} else if flagValue != raw.AdminUsername {
			return fmt.Errorf("%w: both Spec.mke.AdminUsername and Spec.mke.UpgradeFlags --admin-username set, only one allowed", errMKEConfigInvalid)
		}
	}

	if flagValue := raw.UpgradeFlags.GetValue("--admin-password"); flagValue != "" {
		if raw.AdminPassword == "" {
			raw.AdminPassword = flagValue
			raw.UpgradeFlags.Delete("--admin-password")
		} else if flagValue != raw.AdminPassword {
			return fmt.Errorf("%w: both Spec.mke.AdminPassword and Spec.mke.UpgradeFlags --admin-password set, only one allowed", errMKEConfigInvalid)
		}
	}

	if raw.CACertPath != "" {
		caCertData, err := fileutil.LoadExternalFile(raw.CACertPath)
		if err != nil {
			return fmt.Errorf("failed to load CA cert file: %w", err)
		}
		raw.CACertData = string(caCertData)
	}

	if raw.CertPath != "" {
		certData, err := fileutil.LoadExternalFile(raw.CertPath)
		if err != nil {
			return fmt.Errorf("failed to load cert file: %w", err)
		}
		raw.CertData = string(certData)
	}

	if raw.KeyPath != "" {
		keyData, err := fileutil.LoadExternalFile(raw.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to load key file: %w", err)
		}
		raw.KeyData = string(keyData)
	}

	// to make it easier to set in tests
	if c.Version != "" && raw.Version == "" {
		raw.Version = c.Version
	}

	if raw.Version == "" {
		return fmt.Errorf("%w: missing spec.mke.version", errMKEConfigInvalid)
	}

	v, err := version.NewVersion(raw.Version)
	if err != nil {
		return fmt.Errorf("%w: error in field spec.mke.version: %w", errMKEConfigInvalid, err)
	}

	if raw.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		raw.ImageRepo = constant.ImageRepoLegacy
	}

	*c = MKEConfig(raw)
	return nil
}

// NewMKEConfig creates new config with sane defaults.
func NewMKEConfig() MKEConfig {
	return MKEConfig{
		ImageRepo: constant.ImageRepo,
		Metadata:  &MKEMetadata{},
	}
}

// UseLegacyImageRepo returns true if the version number does not satisfy >=3.1.15 || >=3.2.8 || >=3.3.2.
func (c *MKEConfig) UseLegacyImageRepo(mkeVersion *version.Version) bool {
	// Strip out anything after -, seems like go-version thinks
	// 3.1.16-rc1 does not satisfy >= 3.1.15  (nor >= 3.1.15-a)
	vs := mkeVersion.String()
	if idx := strings.Index(vs, "-"); idx >= 0 {
		vBase, err := version.NewVersion(vs[:idx])
		if err == nil {
			mkeVersion = vBase
		}
	}

	constraints := []string{
		"< 3.2, >= 3.1.15",
		"> 3.1, < 3.3, >= 3.2.8",
		"> 3.2, < 3.4, >= 3.3.2",
		">= 3.4",
	}

	for _, cs := range constraints {
		constraint, err := version.NewConstraint(cs)
		if err != nil {
			return false
		}
		if constraint.Check(mkeVersion) {
			return false
		}
	}
	return true
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config.
func (c *MKEConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/ucp:%s", c.ImageRepo, c.Version)
}
