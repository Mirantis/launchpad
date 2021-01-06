package api

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/creasty/defaults"
	"github.com/k0sproject/rig/util"

	"github.com/hashicorp/go-version"
)

// MKEConfig has all the bits needed to configure mke during installation
type MKEConfig struct {
	Version         string       `yaml:"version"`
	ImageRepo       string       `yaml:"imageRepo,omitempty"`
	AdminUsername   string       `yaml:"adminUsername,omitempty"`
	AdminPassword   string       `yaml:"adminPassword,omitempty"`
	InstallFlags    common.Flags `yaml:"installFlags,omitempty,flow"`
	UpgradeFlags    common.Flags `yaml:"upgradeFlags,omitempty,flow"`
	ConfigFile      string       `yaml:"configFile,omitempty" validate:"omitempty,file"`
	ConfigData      string       `yaml:"configData,omitempty"`
	LicenseFilePath string       `yaml:"licenseFilePath,omitempty" validate:"omitempty,file"`
	CACertPath      string       `yaml:"caCertPath,omitempty" validate:"omitempty,file"`
	CertPath        string       `yaml:"certPath,omitempty" validate:"omitempty,file"`
	KeyPath         string       `yaml:"keyPath,omitempty" validate:"omitempty,file"`
	CACertData      string       `yaml:"caCertData,omitempty"`
	CertData        string       `yaml:"certData,omitempty"`
	KeyData         string       `yaml:"keyData,omitempty"`
	Cloud           *MKECloud    `yaml:"cloud,omitempty"`

	Metadata *MKEMetadata `yaml:"-"`
}

// MKEMetadata has the "runtime" discovered metadata of already existing installation.
type MKEMetadata struct {
	Installed               bool
	InstalledVersion        string
	InstalledBootstrapImage string
	ClusterID               string
	VXLAN                   bool
	ManagerJoinToken        string
	WorkerJoinToken         string
}

// MKECloud has the cloud provider configuration
type MKECloud struct {
	Provider   string `yaml:"provider,omitempty" validate:"required"`
	ConfigFile string `yaml:"configFile,omitempty" validate:"omitempty,file"`
	ConfigData string `yaml:"configData,omitempty"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *MKEConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(c)

	type mkeConfig MKEConfig
	yc := (*mkeConfig)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	if c.ConfigFile != "" {
		configData, err := util.LoadExternalFile(c.ConfigFile)
		if err != nil {
			return err
		}
		c.ConfigData = string(configData)
	}

	if c.Cloud != nil && c.Cloud.ConfigFile != "" {
		cloudConfigData, err := util.LoadExternalFile(c.Cloud.ConfigFile)
		if err != nil {
			return err
		}
		c.Cloud.ConfigData = string(cloudConfigData)
	}

	if flagValue := c.InstallFlags.GetValue("--admin-username"); flagValue != "" {
		if c.AdminUsername == "" {
			c.AdminUsername = flagValue
			c.InstallFlags.Delete("--admin-username")
		} else if flagValue != c.AdminUsername {
			return fmt.Errorf("both Spec.mke.AdminUsername and Spec.mke.InstallFlags --admin-username set, only one allowed")
		}
	}

	if flagValue := c.InstallFlags.GetValue("--admin-password"); flagValue != "" {
		if c.AdminPassword == "" {
			c.AdminPassword = flagValue
			c.InstallFlags.Delete("--admin-password")
		} else if flagValue != c.AdminPassword {
			return fmt.Errorf("both Spec.mke.AdminPassword and Spec.mke.InstallFlags --admin-password set, only one allowed")
		}
	}

	if flagValue := c.UpgradeFlags.GetValue("--admin-username"); flagValue != "" {
		if c.AdminUsername == "" {
			c.AdminUsername = flagValue
			c.UpgradeFlags.Delete("--admin-username")
		} else if flagValue != c.AdminUsername {
			return fmt.Errorf("both Spec.mke.AdminUsername and Spec.mke.UpgradeFlags --admin-username set, only one allowed")
		}
	}

	if flagValue := c.UpgradeFlags.GetValue("--admin-password"); flagValue != "" {
		if c.AdminPassword == "" {
			c.AdminPassword = flagValue
			c.UpgradeFlags.Delete("--admin-password")
		} else if flagValue != c.AdminPassword {
			return fmt.Errorf("both Spec.mke.AdminPassword and Spec.mke.UpgradeFlags --admin-password set, only one allowed")
		}
	}

	if c.CACertPath != "" {
		caCertData, err := util.LoadExternalFile(c.CACertPath)
		if err != nil {
			return err
		}
		c.CACertData = string(caCertData)
	}

	if c.CertPath != "" {
		certData, err := util.LoadExternalFile(c.CertPath)
		if err != nil {
			return err
		}
		c.CertData = string(certData)
	}

	if c.KeyPath != "" {
		keyData, err := util.LoadExternalFile(c.KeyPath)
		if err != nil {
			return err
		}
		c.KeyData = string(keyData)
	}

	v, err := version.NewVersion(c.Version)
	if err != nil {
		return err
	}

	if c.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		c.ImageRepo = constant.ImageRepoLegacy
	}

	return nil
}

// SetDefaults implements the defaults package Setter interface
func (c *MKEConfig) SetDefaults() {
	if defaults.CanUpdate(c.Version) {
		c.Version = constant.MKEVersion
	}

	if defaults.CanUpdate(c.ImageRepo) {
		c.ImageRepo = constant.ImageRepo
	}

	if defaults.CanUpdate(c.Metadata) {
		c.Metadata = &MKEMetadata{}
	}
}

// UseLegacyImageRepo returns true if the version number does not satisfy >=3.1.15 || >=3.2.8 || >=3.3.2
func (c *MKEConfig) UseLegacyImageRepo(v *version.Version) bool {

	// Strip out anything after -, seems like go-version thinks
	// 3.1.16-rc1 does not satisfy >= 3.1.15  (nor >= 3.1.15-a)
	vs := v.String()
	var v2 *version.Version
	if strings.Contains(vs, "-") {
		v2, _ = version.NewVersion(vs[0:strings.Index(vs, "-")])
	} else {
		v2 = v
	}

	c1, _ := version.NewConstraint("< 3.2, >= 3.1.15")
	c2, _ := version.NewConstraint("> 3.1, < 3.3, >= 3.2.8")
	c3, _ := version.NewConstraint("> 3.3, < 3.4, >= 3.3.2")
	c4, _ := version.NewConstraint(">= 3.4")
	return !(c1.Check(v2) || c2.Check(v2) || c3.Check(v2) || c4.Check(v2))
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config
func (c *MKEConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/ucp:%s", c.ImageRepo, c.Version)
}
