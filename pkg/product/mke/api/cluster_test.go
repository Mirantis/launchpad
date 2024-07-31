package api

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Mirantis/mcc/pkg/config/migration"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v11"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v12"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v13"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v14"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v15"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta1"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta2"
	// needed to load the migrators.
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta3"
	"github.com/Mirantis/mcc/pkg/constant"
	validator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestHostRequireManagerValidationPass(t *testing.T) {
	kf, _ := os.CreateTemp("", "testkey")
	defer kf.Close()
	data := `
apiVersion: "launchpad.mirantis.com/mke/v1.6"
kind: mke
spec:
  hosts:
    - ssh:
        address: 10.0.0.1
				keyPath: ` + kf.Name() + `
      role: manager
    - ssh:
        address: 10.0.0.2
				keyPath: ` + kf.Name() + `
      role: worker
	mke:
	  version: 3.3.7
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.NoError(t, err)
}

func TestHostRequireManagerValidationFail(t *testing.T) {
	kf, _ := os.CreateTemp("", "testkey")
	defer kf.Close()
	data := `
apiVersion: "launchpad.mirantis.com/mke/v1.4"
kind: mke
spec:
  hosts:
    - ssh:
        address: 10.0.0.1
				keyPath: ` + kf.Name() + `
      role: worker
    - ssh:
        address: 10.0.0.2
				keyPath: ` + kf.Name() + `
      role: worker
	mke:
	  version: 3.3.7
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)

	validateErrorField(t, err, "hosts")
}

func TestNonExistingHostsFails(t *testing.T) {
	data := `
apiVersion: "launchpad.mirantis.com/mke/v1.4"
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)

	validateErrorField(t, err, "Hosts")
}

func TestHostAddressValidationWithInvalidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
  - ssh:
      address: "512.1.2.3.@"
    role: manager
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Address")
}

func TestHostAddressValidationWithValidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
  - ssh:
      address: "10.10.10.10"
    role: manager
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.NotContains(t, getAllErrorFields(err), "Address")
}

func TestHostAddressValidationWithInvalidHostname(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - ssh:
        address: "1-2-foo.@"
      role: manager
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Address")
}

func TestHostAddressValidationWithValidHostname(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - ssh:
        address: "foo.example.com"
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.NotContains(t, getAllErrorFields(err), "Address")

}

func TestHostSshPortValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - ssh:
        address: "1.2.3.4"
        port: 65537
      role: manager
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Port")
}

func TestHostRoleValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
  - address: "1.2.3.4"
    ssh:
      port: 22
    role: foobar
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Role")
}

func TestHostWithComplexMCRConfig(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
  - ssh:
      address: "1.2.3.4"
      port: 22
    role: worker
    mcrConfig:
      debug: true
      log-opts:
        max-size: 10m
        max-files: 5
`
	c := loadYaml(t, data)

	_, err := json.Marshal(c.Spec.Hosts[0].DaemonConfig)
	require.NoError(t, err)
}

func TestMigrateFromV1Beta1(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: mke
spec:
	ucp:
	  version: 3.3.7
  engine:
    installURL: http://example.com/
  hosts:
  - address: "1.2.3.4"
    sshPort: 9022
    sshKeyPath: /path/to/nonexisting
    user: foofoo
    role: manager
`
	c := loadAndMigrateYaml(t, data)
	err := c.Validate()
	require.NoError(t, err)
	require.Equal(t, "launchpad.mirantis.com/mke/v1.6", c.APIVersion)

	require.Equal(t, c.Spec.MCR.InstallURLLinux, "http://example.com/")
	require.Equal(t, c.Spec.Hosts[0].SSH.Port, 9022)
	require.Equal(t, c.Spec.Hosts[0].SSH.User, "foofoo")
}

func TestMigrateFromV1Beta2(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta2
kind: mke
spec:
  ucp:
	  version: 3.3.7
  engine:
    installURL: http://example.com/
  hosts:
  - address: "1.2.3.4"
    role: manager
    winRM:
      user: foo
      password: foo
`
	c := loadAndMigrateYaml(t, data)
	require.NoError(t, c.Validate())
	require.Equal(t, "launchpad.mirantis.com/mke/v1.6", c.APIVersion)
}

func TestMigrateFromV1Beta1WithoutInstallURL(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: mke
spec:
	ucp:
	  version: 3.3.7
  engine:
    version: 1.2.3
  hosts:
  - address: "1.2.3.4"
    sshPort: 9022
    sshKeyPath: /path/to/nonexisting
    user: foofoo
    role: manager
`
	c := loadAndMigrateYaml(t, data)
	err := c.Validate()
	require.NoError(t, err)
	require.Equal(t, "launchpad.mirantis.com/mke/v1.6", c.APIVersion)

	require.Equal(t, constant.MCRInstallURLLinux, c.Spec.MCR.InstallURLLinux)
	require.Equal(t, 9022, c.Spec.Hosts[0].SSH.Port)
	require.Equal(t, "foofoo", c.Spec.Hosts[0].SSH.User)
}

func TestHostWinRMCACertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - role: manager
      winRM:
        address: "10.0.0.1"
        caCertPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "CACertPath")
}

func TestHostWinRMCertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - role: manager
      winRM:
        address: "10.0.0.1"
        certPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "CertPath")
}

func TestHostWinRMKeyPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - role: manager
      winRM:
        keyPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
}

func TestHostSSHDefaults(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - ssh:
        address: "1.2.3.4"
      role: manager
`
	c := loadYaml(t, data)

	require.Equal(t, c.Spec.Hosts[0].SSH.User, "root")
	require.Equal(t, c.Spec.Hosts[0].SSH.Port, 22)
}

func TestHostWinRMDefaults(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
	mke:
	  version: 3.3.7
  hosts:
    - role: manager
      winRM:
        address: 10.0.0.1
        user: User
`
	c := loadYaml(t, data)

	require.NoError(t, c.Validate())

	require.Equal(t, c.Spec.Hosts[0].WinRM.User, "User")
	require.Equal(t, c.Spec.Hosts[0].WinRM.Port, 5985)
	require.Equal(t, c.Spec.Hosts[0].WinRM.UseNTLM, false)
	require.Equal(t, c.Spec.Hosts[0].WinRM.UseHTTPS, false)
	require.Equal(t, c.Spec.Hosts[0].WinRM.Insecure, false)
}

func TestValidationWithMSR2Role(t *testing.T) {
	kf, _ := os.CreateTemp("", "testkey")
	defer kf.Close()
	t.Run("the role is not ucp, worker or msr2", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
	mke:
	  version: 3.3.7
	msr:
	  version: 2.8.5
  hosts:
    - ssh:
        address: "10.0.0.1"
				keyPath: ` + kf.Name() + `
      role: weirdrole
    - ssh:
        address: "10.0.0.2"
      role: manager
`
		c := loadYaml(t, data)
		require.Error(t, c.Validate())
	})

	t.Run("the role is msr2", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke+msr
spec:
	mke:
	  version: 3.3.7
	msr2:
	  version: 2.8.5
  hosts:
    - ssh:
        address: "10.0.0.1"
				keyPath: ` + kf.Name() + `
      role: msr2
    - ssh:
        address: "10.0.0.2"
				keyPath: ` + kf.Name() + `
      role: manager
`
		c := loadYaml(t, data)
		require.NoError(t, c.Validate())
	})

}

func TestValidationWithMSR3(t *testing.T) {
	t.Run("crd is a valid unstructured object", func(t *testing.T) {
		// There is an extra tab in the yaml data
		// under crd.apiVersion, for whatever
		// reason the loadYaml helper is not
		// handling the tabs correctly for that
		// section.  The resulting data is valid
		// yaml.
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke+msr
spec:
	mke:
	  version: 3.3.7
	msr3:
	  version: 3.1.4
	  storageURL: "https://example.com"
	  storageClassType: "nfs"
	  crd:
	    apiVersion: "msr.mirantis.com/v1"
		  kind: "MSR"
		  spec:
		    logLevel: "debug"
  hosts:
    - ssh:
        address: "10.0.0.1"
      role: msr3
    - ssh:
        address: "10.0.0.2"
      role: manager
`
		c := loadYaml(t, data)

		require.Equal(t, c.Spec.MSR3.StorageURL, "https://example.com")
		require.Equal(t, c.Spec.MSR3.StorageClassType, "nfs")
		require.Equal(t, c.Spec.MSR3.CRD.GetAPIVersion(), "msr.mirantis.com/v1")
		require.Equal(t, c.Spec.MSR3.CRD.GetKind(), "MSR")

		actual, found, err := unstructured.NestedString(c.Spec.MSR3.CRD.Object, "spec", "logLevel")
		require.True(t, found)
		require.NoError(t, err)
		require.Equal(t, actual, "debug")

		require.NoError(t, c.Validate())
	})

	t.Run("storageURL is required when storageClassType is set", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke+msr
spec:
	mke:
	  version: 3.3.7
	msr3:
	  version: 3.1.4
	  storageClassType: "nfs"
  hosts:
    - ssh:
        address: "10.0.0.1"
      role: msr3
    - ssh:
        address: "10.0.0.2"
      role: manager
`

		c := &ClusterConfig{}
		err := yaml.Unmarshal([]byte(strings.ReplaceAll(data, "\t", "  ")), c)
		require.ErrorContains(t, err, "spec.msr.storageURL", "required")
	})

	t.Run("storageClassType must be one of the supported types", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke+msr
spec:
	mke:
	  version: 3.3.7
	msr3:
	  version: 3.1.4
	  storageURL: "https://example.com"
	  storageClassType: "not-supported"
  hosts:
    - ssh:
        address: "10.0.0.1"
      role: msr3
    - ssh:
        address: "10.0.0.2"
      role: manager
`

		c := loadYaml(t, data)
		require.ErrorContains(t, c.Validate(), "StorageClassType", "oneof")
	})
}

// Just a small helper to load the config struct from yaml to get defaults etc. in place.
func loadYaml(t *testing.T, data string) *ClusterConfig {
	c := &ClusterConfig{}
	// convert any tabs added by editor into double spaces
	require.NoError(t, yaml.Unmarshal([]byte(strings.ReplaceAll(data, "\t", "  ")), c))

	return c
}

// Just a small helper to load the config struct from yaml through the migrations.
func loadAndMigrateYaml(t *testing.T, data string) *ClusterConfig {
	c := &ClusterConfig{}
	raw := make(map[string]interface{})
	// convert any tabs added by editor into double spaces
	require.NoError(t, yaml.Unmarshal([]byte(strings.ReplaceAll(data, "\t", "  ")), raw))
	require.NoError(t, migration.Migrate(raw))
	newdata, err := yaml.Marshal(raw)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(newdata, c))
	return c
}

// checks that the validation errors contains error for the expected field.
func validateErrorField(t *testing.T, err error, field string) {
	fields := getAllErrorFields(err)
	require.Contains(t, fields, field)
}

func getAllErrorFields(err error) []string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return nil
	}
	fields := make([]string, len(validationErrors))

	// Collect all fields that failed validation
	// Also "store" the validation error for the expected field so that we can return it
	// and the correcponding test can further validate it if needed
	for i, fieldError := range validationErrors {
		fields[i] = fieldError.Field()
	}

	return fields
}
