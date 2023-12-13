package api

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/helm"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util/fileutil"
	"github.com/creasty/defaults"
	"github.com/hashicorp/go-version"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type MSRConfig struct {
	Version string `yaml:"version" validate:"required"`

	// V2 defines the configuration for MSR V2.
	V2 MSR2Config `yaml:",inline"`
	// V3 defines the configuration for MSR V3.
	V3 MSR3Config `yaml:",inline"`
}

// MSRConfig has all the bits needed to configure MSR (V2) during installation.
type MSR2Config struct {
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

// MSR3Config defines the configuration for both the MSR3 CR and the
// dependencies needed to run it.
type MSR3Config struct {
	// Dependencies define strict dependencies that MSR3 needs to function.
	Dependencies `yaml:"dependencies,omitempty"`
	// StorageClassType allows users to have launchpad configure a StorageClass
	// on their behalf and set the target cluster to use that as the default.
	StorageClassType string `yaml:"storageClassType,omitempty" validate:"omitempty,oneof=nfs"`
	// StorageURL defines the URL that StorageClassType will use when
	// configuring.  It is required when StorageClassType is specified.
	StorageURL string `yaml:"storageURL,omitempty"`
	// LoadBalancerURL allows users to have launchpad expose MSR3 with a
	// default configuration of LoadBalancer type.
	LoadBalancerURL string `yaml:"loadBalancerURL,omitempty"`
	// CRD is the MSR custom resource definition which configures the MSR CR.
	CRD *unstructured.Unstructured `yaml:"crd,omitempty"`
}

func (c *MSR3Config) Validate() error {
	if c.StorageClassType != "" && c.StorageURL == "" {
		return fmt.Errorf("spec.msr.storageURL is required when spec.msr.storageClassType is populated")
	}

	return nil
}

func (c *MSR3Config) ShouldConfigureStorageClass() bool {
	return c.StorageClassType != "" && c.StorageURL != ""
}

func (c *MSR3Config) ShouldConfigureLB() bool {
	return c.LoadBalancerURL != ""
}

// Dependencies define strict dependencies that MSR3 needs to function.
type Dependencies struct {
	CertManager       *helm.ReleaseDetails `yaml:"certManager"`
	PostgresOperator  *helm.ReleaseDetails `yaml:"postgresOperator"`
	RethinkDBOperator *helm.ReleaseDetails `yaml:"rethinkDBOperator"`
	MSROperator       *helm.ReleaseDetails `yaml:"msrOperator"`
}

// List returns a list of all dependencies from the MSR3Config.
func (d *Dependencies) List() []*helm.ReleaseDetails {
	return []*helm.ReleaseDetails{
		d.CertManager,
		d.PostgresOperator,
		d.RethinkDBOperator,
		d.MSROperator,
	}
}

// SetDefaults populates unset fields with sane defaults.
func (d *Dependencies) SetDefaults() {
	if d.CertManager == nil {
		d.CertManager = &helm.ReleaseDetails{}

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
		d.PostgresOperator = &helm.ReleaseDetails{}

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
				"configKubernetes": map[string]int{
					"spilo_runasuser":  101,
					"spilo_runasgroup": 103,
					"spilo_fsgroup":    103,
				},
			}
		}
	}

	if d.RethinkDBOperator == nil {
		d.RethinkDBOperator = &helm.ReleaseDetails{}

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
			d.RethinkDBOperator.Version = "1.0.1"
		}
	}

	if d.MSROperator == nil {
		d.MSROperator = &helm.ReleaseDetails{}

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
			d.MSROperator.Version = "1.0.1"
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

	if err := defaults.Set(c); err != nil {
		return err
	}

	return c.setConfigForVersion()
}

func (c *MSRConfig) setConfigForVersion() error {
	switch c.MajorVersion() {
	case 2:
		if c.V2.CACertPath != "" {
			caCertData, err := fileutil.LoadExternalFile(c.V2.CACertPath)
			if err != nil {
				return err
			}
			c.V2.CACertData = string(caCertData)
		}

		if c.V2.CertPath != "" {
			certData, err := fileutil.LoadExternalFile(c.V2.CertPath)
			if err != nil {
				return err
			}
			c.V2.CertData = string(certData)
		}

		if c.V2.KeyPath != "" {
			keyData, err := fileutil.LoadExternalFile(c.V2.KeyPath)
			if err != nil {
				return err
			}
			c.V2.KeyData = string(keyData)
		}

	case 3:
		if err := c.V3.Validate(); err != nil {
			return fmt.Errorf("failed to validate MSR configuration: %w", err)
		}

		c.V3.Dependencies.SetDefaults()

	default:
		return fmt.Errorf("unsupported MSR major version: must be either 2 or 3")
	}

	return nil
}

// MajorVersion returns the major version of MSR, or 0 if the version is invalid.
func (c *MSRConfig) MajorVersion() int {
	if c == nil {
		return 0
	}

	v, err := version.NewVersion(c.Version)
	if err != nil {
		return 0
	}

	return v.Segments()[0]
}

// SetDefaults sets default values.
func (c *MSRConfig) SetDefaults() {
	if c.V2.ImageRepo == "" {
		c.V2.ImageRepo = constant.ImageRepo
	}

	v, err := version.NewVersion(c.Version)
	if err != nil {
		return
	}

	if c.V2.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		c.V2.ImageRepo = constant.ImageRepoLegacy
	}
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config.
func (c *MSRConfig) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/dtr:%s", c.V2.ImageRepo, c.Version)
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
