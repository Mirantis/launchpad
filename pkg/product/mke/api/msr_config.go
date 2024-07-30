package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/helm"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util/fileutil"
	"github.com/Mirantis/mcc/pkg/util/maputil"
	"github.com/creasty/defaults"
	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// MSR2Config has all the bits needed to configure MSR (V2) during installation.
type MSR2Config struct {
	Version      string       `yaml:"version,omitempty"`
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
	// Version is the MSR3 version to install.
	Version string `yaml:"version,omitempty"`
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

	Metadata MSR3Metadata `yaml:"-"`
}

type MSR3Metadata struct {
	Installed        bool
	InstalledVersion string
	// InstalledDependencies is a map of dependencies needed for MSR3 and their
	// versions.
	InstalledDependencies map[string]helm.ReleaseDetails
}

var errStorageURLRequired = fmt.Errorf("spec.msr.storageURL is required when spec.msr.storageClassType is populated")

func (c *MSR3Config) Validate() error {
	if c.StorageClassType != "" && c.StorageURL == "" {
		return errStorageURLRequired
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
			d.CertManager.Version = "1.14.7"
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
			d.PostgresOperator.Version = "1.12.2"
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

var errUnexpectedTypeForCRD = errors.New("unexpected type for CRD: expected map")

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *MSR3Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type msr3 MSR3Config
	yc := (*msr3)(c)
	if err := unmarshal(yc); err != nil {
		return fmt.Errorf("failed to unmarshal MSR configuration: %w", err)
	}

	// Decode the MSR configuration into a map, check to see if the CRD field
	// is present and if so use the decoded map to form the unstructured object.
	var obj map[interface{}]interface{}
	if err := unmarshal(&obj); err != nil {
		return fmt.Errorf("failed to unmarshal MSR configuration into map: %w", err)
	}

	if _, ok := obj["crd"]; ok {
		// Due to yaml.v2's unmarshalling behavior, we first need to unmarshal
		// into map[interface{}]interface{} and then into map[string]interface{}.
		// If we move to yaml.v3 in the future, we can Decode directly into
		// map[string]interface{} and gain some performance improvements around
		// this conversion.
		mapObj, ok := obj["crd"].(map[interface{}]interface{})
		if !ok {
			return errUnexpectedTypeForCRD
		}

		// Convert the map[interface{}]interface{} to map[string]interface{}.
		c.CRD = &unstructured.Unstructured{Object: maputil.ConvertInterfaceMap(mapObj)}
	}

	if err := defaults.Set(c); err != nil {
		return fmt.Errorf("failed to set defaults for MSR configuration: %w", err)
	}

	if err := c.Validate(); err != nil {
		return fmt.Errorf("failed to validate MSR configuration: %w", err)
	}

	c.Dependencies.SetDefaults()

	return nil
}

// SetDefaults sets default values.
func (c *MSR2Config) SetDefaults() error {
	if c.CACertPath != "" {
		caCertData, err := fileutil.LoadExternalFile(c.CACertPath)
		if err != nil {
			return fmt.Errorf("failed to load CA cert data: %w", err)
		}
		c.CACertData = string(caCertData)
	}

	if c.CertPath != "" {
		certData, err := fileutil.LoadExternalFile(c.CertPath)
		if err != nil {
			return fmt.Errorf("failed to load cert data: %w", err)
		}
		c.CertData = string(certData)
	}

	if c.KeyPath != "" {
		keyData, err := fileutil.LoadExternalFile(c.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to load key data: %w", err)
		}
		c.KeyData = string(keyData)
	}

	if c.ImageRepo == "" {
		c.ImageRepo = constant.ImageRepo

		fmt.Println("ImageRepo is empty")
	}

	v, err := version.NewVersion(c.Version)
	if err != nil {
		log.Debugf("Failed to parse version: %s, will fallback to using imageRepo: %s", c.Version, constant.ImageRepo)
		// If we encounter an error here just default to using
		// constant.ImageRepo.
		return nil
	}

	if c.ImageRepo == constant.ImageRepo && c.UseLegacyImageRepo(v) {
		c.ImageRepo = constant.ImageRepoLegacy
	}

	return nil
}

// GetBootstrapperImage combines the bootstrapper image name based on user given config.
func (c *MSR2Config) GetBootstrapperImage() string {
	return fmt.Sprintf("%s/dtr:%s", c.ImageRepo, c.Version)
}

// UseLegacyImageRepo returns true if the version number does not satisfy >= 2.8.2 || >= 2.7.8 || >= 2.6.15.
func (c *MSR2Config) UseLegacyImageRepo(v *version.Version) bool {
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

var errInvalidMSR2Config = fmt.Errorf("invalid MSR2 config")

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *MSR2Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type msr2 MSR2Config
	yc := (*msr2)(c)
	if err := unmarshal(yc); err != nil {
		return err
	}

	if c.Version == "" {
		return fmt.Errorf("%w: missing spec.msr.version", errInvalidMSR2Config)
	}

	if _, err := version.NewVersion(c.Version); err != nil {
		return fmt.Errorf("%w: error in field spec.msr.version: %w", errInvalidMSR2Config, err)
	}

	if err := defaults.Set(c); err != nil {
		return fmt.Errorf("set MSR2 defaults: %w", err)
	}

	return c.SetDefaults()
}
