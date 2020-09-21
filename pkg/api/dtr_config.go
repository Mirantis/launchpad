package api

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/hashicorp/go-version"
)

// DtrConfig has all the bits needed to configure DTR during installation
type DtrConfig struct {
	Version       string   `yaml:"version"`
	ImageRepo     string   `yaml:"imageRepo,omitempty"`
	InstallFlags  []string `yaml:"installFlags,flow,omitempty"`
	ReplicaConfig string   `yaml:"replicaConfig,omitempty"  default:"random"`

	Metadata *DtrMetadata `yaml:"-"`
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

	v, err := version.NewVersion(raw.Version)
	if err != nil {
		return err
	}

	if raw.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		raw.ImageRepo = constant.ImageRepoLegacy
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

// UseLegacyImageRepo returns true if the version number does not satisfy >= 2.8.2 || >= 2.7.8 || >= 2.6.15
func (c *DtrConfig) UseLegacyImageRepo(v *version.Version) bool {

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
