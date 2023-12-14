package api

import (
	"fmt"
	"strings"

	msrv1 "github.com/Mirantis/msr-operator/api/v1"
	"github.com/creasty/defaults"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/helm"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
)

// MSRConfig has all the bits needed to configure MSR during installation.
type MSRConfig struct {
	Version string `yaml:"version" validate:"required"`

	// These fields configure the MSR3 CR managed by msr-operator. These
	// value's cannot be used to configure MSR.
	*MSR3Config `yaml:"msr3,omitempty"`

	// These fields configure the MSR installer.  These cannot be used
	// to configure MSR3 but are left here to support legacy configs.
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

var errInvalidMSRConfig = fmt.Errorf("invalid MSR config")

// MSR3Config defines the configuration for both the MSR3 CR and the
// dependencies needed to run it.
type MSR3Config struct {
	Dependencies `yaml:"dependencies,omitempty"`
	msrv1.MSR    `yaml:"spec,omitempty"`
}

type Dependencies struct {
	CertManager       *helm.ChartDetails `yaml:"certManager"`
	PostgresOperator  *helm.ChartDetails `yaml:"postgresOperator"`
	RethinkDBOperator *helm.ChartDetails `yaml:"rethinkDBOperator"`
	MSROperator       *helm.ChartDetails `yaml:"msrOperator"`
}

// List returns a list of all dependencies from the MSR3Config.
func (d *Dependencies) List() []*helm.ChartDetails {
	return []*helm.ChartDetails{
		d.CertManager,
		d.PostgresOperator,
		d.RethinkDBOperator,
		d.MSROperator,
	}
}

// SetDefaults populates unset fields with sane defaults.
// FIXME: Maybe we should just use default struct tags instead of this?
func (d *Dependencies) SetDefaults() {
	if d.CertManager == nil {
		d.CertManager = &helm.ChartDetails{}

		if d.CertManager.ChartName == "" {
			d.CertManager.ChartName = constant.CertManager
		}
		if d.CertManager.ReleaseName == "" {
			d.CertManager.ReleaseName = constant.CertManager
		}
		if d.CertManager.RepoURL == "" {
			d.CertManager.RepoURL = "https://charts.jetstack.io"
		}
		if d.CertManager.Version == "" {
			d.CertManager.Version = "1.12.4"
		}
		if d.CertManager.Values == nil {
			d.CertManager.Values = map[string]interface{}{"installCRDs": true}
		}
	}

	if d.PostgresOperator == nil {
		d.PostgresOperator = &helm.ChartDetails{}

		if d.PostgresOperator.ChartName == "" {
			d.PostgresOperator.ChartName = constant.PostgresOperator
		}
		if d.PostgresOperator.ReleaseName == "" {
			d.PostgresOperator.ReleaseName = constant.PostgresOperator
		}
		if d.PostgresOperator.RepoURL == "" {
			d.PostgresOperator.RepoURL = "https://opensource.zalando.com/postgres-operator/charts/postgres-operator"
		}
		if d.PostgresOperator.Version == "" {
			d.PostgresOperator.Version = "1.10.0"
		}
		if d.PostgresOperator.Values == nil {
			d.PostgresOperator.Values = map[string]interface{}{
				"configKubernetes.spilo_runasuser":  "101",
				"configKubernetes.spilo_runasgroup": "103",
				"configKubernetes.spilo_fsgroup":    "103",
			}
		}
	}

	if d.RethinkDBOperator == nil {
		d.RethinkDBOperator = &helm.ChartDetails{}

		if d.RethinkDBOperator.ChartName == "" {
			d.RethinkDBOperator.ChartName = constant.RethinkDBOperator
		}
		if d.RethinkDBOperator.ReleaseName == "" {
			d.RethinkDBOperator.ReleaseName = constant.RethinkDBOperator
		}
		if d.RethinkDBOperator.RepoURL == "" {
			d.RethinkDBOperator.RepoURL = "https://registry.mirantis.com/charts/rethinkdb/rethinkdb-operator"
		}
		if d.RethinkDBOperator.Version == "" {
			d.RethinkDBOperator.Version = "1.0.0"
		}
	}

	if d.MSROperator == nil {
		d.MSROperator = &helm.ChartDetails{}

		if d.MSROperator.ChartName == "" {
			d.MSROperator.ChartName = constant.MSROperator
		}
		if d.MSROperator.ReleaseName == "" {
			d.MSROperator.ReleaseName = constant.MSROperator
		}
		if d.MSROperator.RepoURL == "" {
			d.MSROperator.RepoURL = "https://registry.mirantis.com/charts/msr/msr-operator"
		}
		if d.MSROperator.Version == "" {
			d.MSROperator.Version = "1.0.0"
		}
	}
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *MSRConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type msr MSRConfig
	yc := (*msr)(c)
	if err := unmarshal(yc); err != nil {
		return err
	}

	if c.Version == "" {
		return fmt.Errorf("%w: missing spec.msr.version", errInvalidMSRConfig)
	}

	v, err := version.NewVersion(c.Version)
	if err != nil {
		return fmt.Errorf("%w: error in field spec.msr.version: %w", errInvalidMSRConfig, err)
	}

	if err := c.ValidateConfigForMajorVersion(v); err != nil {
		return err
	}

	if c.CACertPath != "" {
		caCertData, err := util.LoadExternalFile(c.CACertPath)
		if err != nil {
			return fmt.Errorf("failed to load msr ca cert file: %w", err)
		}
		c.CACertData = string(caCertData)
	}

	if c.CertPath != "" {
		certData, err := util.LoadExternalFile(c.CertPath)
		if err != nil {
			return fmt.Errorf("failed to load msr cert file: %w", err)
		}
		c.CertData = string(certData)
	}

	if c.KeyPath != "" {
		keyData, err := util.LoadExternalFile(c.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to load msr key file: %w", err)
		}
		c.KeyData = string(keyData)
	}

	c.Dependencies.SetDefaults()

	if err := defaults.Set(c); err != nil {
		return fmt.Errorf("failed to set MSR defaults: %w", err)
	}

	return nil
}

// MajorVersion returns the major version of MSR, or 0 if the version is invalid.
func (c *MSRConfig) MajorVersion() int {
	v, err := version.NewVersion(c.Version)
	if err != nil {
		return 0
	}

	return v.Segments()[0]
}

// ValidateConfigForMajorVersion validates the MSRConfig, ensuring the provided
// fields are valid for the major MSR version provided.
func (c *MSRConfig) ValidateConfigForMajorVersion(v *version.Version) error {
	switch c.MajorVersion() {
	case 2:
		if c.MSR3Config != nil {
			return fmt.Errorf("cannot use spec.msr.msr3 to configure MSR version %s", v.String())
		}
	case 3:
		if c.MSR3Config == nil {
			return fmt.Errorf("missing spec.msr.msr3 to configure MSR version %s", v.String())
		}

		if c.ImageRepo != "" || c.ReplicaIDs != "" ||
			c.InstallFlags != nil || c.UpgradeFlags != nil ||
			c.CACertPath != "" || c.CertPath != "" || c.KeyPath != "" ||
			c.CACertData != "" || c.CertData != "" || c.KeyData != "" {
			log.Warnf("ignoring legacy MSR 2.x configuration fields for MSR version %s, only spec.msr.msr3 fields will be used as configuration", v.String())
		}
	default:
		return fmt.Errorf("unknown MSR version: %s, can only manage 2.x and 3.x versions of MSR", v.String())
	}

	return nil
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
