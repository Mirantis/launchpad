package api

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/creasty/defaults"
	"github.com/hashicorp/go-version"
)

// MSRConfig has all the bits needed to configure MSR during installation.
type MSRConfig struct {
	Version      string       `yaml:"version" validate:"required"`
	ImageRepo    string       `yaml:"imageRepo,omitempty"`
	InstallFlags common.Flags `yaml:"installFlags,flow,omitempty"`
	UpgradeFlags common.Flags `yaml:"upgradeFlags,flow,omitempty"`
	ReplicaIDs   string       `yaml:"replicaIDs,omitempty" default:"random"`
	CACertPath   string       `yaml:"caCertPath,omitempty" validate:"omitempty,file"`
	CertPath     string       `yaml:"certPath,omitempty" validate:"omitempty,file"`
	KeyPath      string       `yaml:"keyPath,omitempty" validate:"omitempty,file"`
	CACertData   string       `yaml:"caCertData,omitempty"`
	CertData     string       `yaml:"certData,omitempty"`
	KeyData      string       `yaml:"keyData,omitempty"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *MSRConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type msr MSRConfig
	yc := (*msr)(c)
	if err := unmarshal(yc); err != nil {
		return err
	}

	if c.Version == "" {
		return fmt.Errorf("missing spec.msr.version")
	}

	if _, err := version.NewVersion(c.Version); err != nil {
		return fmt.Errorf("error in field spec.msr.version: %s", err.Error())
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

	return defaults.Set(c)
}

// SetDefaults sets default values.
func (c *MSRConfig) SetDefaults() {
	if c.ImageRepo == "" {
		c.ImageRepo = constant.ImageRepo
	}

	v, err := version.NewVersion(c.Version)
	if err != nil {
		return
	}

	if c.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		c.ImageRepo = constant.ImageRepoLegacy
	}
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config.
func (c *MSRConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/dtr:%s", c.ImageRepo, c.Version)
}

// UseLegacyImageRepo returns true if the version number does not satisfy >= 2.8.2 || >= 2.7.8 || >= 2.6.15.
func (c *MSRConfig) UseLegacyImageRepo(v *version.Version) bool {

	// Strip out anything after -, seems like go-version thinks
	vs := v.String()
	var v2 *version.Version
	if strings.Contains(vs, "-") {
		v2, _ = version.NewVersion(vs[0:strings.Index(vs, "-")])
	} else {
		v2 = v
	}

	c1, _ := version.NewConstraint(">= 2.8.2")
	c2, _ := version.NewConstraint("< 2.8, >= 2.7.8")
	c3, _ := version.NewConstraint("< 2.7, >= 2.6.15")
	return !(c1.Check(v2) || c2.Check(v2) || c3.Check(v2))
}
