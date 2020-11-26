package api

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/hashicorp/go-version"
)

// MSRConfig has all the bits needed to configure MSR during installation
type MSRConfig struct {
	Version      string `yaml:"version"`
	ImageRepo    string `yaml:"imageRepo,omitempty"`
	InstallFlags Flags  `yaml:"installFlags,flow,omitempty"`
	ReplicaIDs   string `yaml:"replicaIDs,omitempty"  default:"random"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *MSRConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type msr MSRConfig
	config := NewMSRConfig()
	raw := msr(config)
	if err := unmarshal(&raw); err != nil {
		return err
	}

	v, err := version.NewVersion(raw.Version)
	if err != nil {
		return err
	}

	if raw.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		raw.ImageRepo = constant.ImageRepoLegacy
	}

	*c = MSRConfig(raw)
	return nil
}

// NewMSRConfig creates new config with sane defaults
func NewMSRConfig() MSRConfig {
	return MSRConfig{
		Version:   constant.MSRVersion,
		ImageRepo: constant.ImageRepo,
	}
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config
func (c *MSRConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/dtr:%s", c.ImageRepo, c.Version)
}

// UseLegacyImageRepo returns true if the version number does not satisfy >= 2.8.2 || >= 2.7.8 || >= 2.6.15
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
