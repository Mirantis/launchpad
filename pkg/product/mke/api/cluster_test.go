package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Mirantis/mcc/pkg/config/migration"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta1"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta2"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1beta3"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v1"
	// needed to load the migrators
	_ "github.com/Mirantis/mcc/pkg/config/migration/v11"
	"github.com/Mirantis/mcc/pkg/constant"
	validator "github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"

	"github.com/stretchr/testify/require"
)

func TestHostRequireManagerValidationPass(t *testing.T) {
	data := `
apiVersion: "launchpad.mirantis.com/mke/v1.2"
kind: mke
spec:
  hosts:
    - address: 10.0.0.1
      localhost: true
      role: manager
    - address: 10.0.0.2
      role: worker
      localhost: true
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.NoError(t, err)
}

func TestHostRequireManagerValidationFail(t *testing.T) {
	data := `
apiVersion: "launchpad.mirantis.com/mke/v1.2"
kind: mke
spec:
  hosts:
    - address: 10.0.0.1
      role: worker
      localhost: true
    - address: 10.0.0.2
      role: worker
      localhost: true
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)

	validateErrorField(t, err, "hosts")
}

func TestNonExistingHostsFails(t *testing.T) {
	data := `
apiVersion: "launchpad.mirantis.com/mke/v1.2"
kind: mke
spec:
  hosts:
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)

	validateErrorField(t, err, "Hosts")
}

func TestHostAddressValidationWithInvalidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "512.1.2.3"
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Address")
}

func TestHostAddressValidationWithValidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "10.10.10.10"
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.NotContains(t, getAllErrorFields(err), "Address")
}

func TestHostAddressValidationWithInvalidHostname(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1-2-foo"
  mke:
    username: foofoo
    password: barbar
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
  hosts:
    - address: "foo.example.com"
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.NotContains(t, getAllErrorFields(err), "Address")

}

func TestHostSshPortValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: manager
      ssh:
        port: 0
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Port")
}

func TestHostSshKeyValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      ssh:
        port: 22
        keyPath: /path/to/nonexisting/key
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
}

func TestHostRoleValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
  - address: "1.2.3.4"
    ssh:
      port: 22
    role: foobar
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "Role")
}

func TestHostWithComplexMCRConfig(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
  - address: "1.2.3.4"
    ssh:
      port: 22
    role: worker
    mcrConfig:
      debug: true
      log-opts:
        max-size: 10m
        max-files: 5
  mke:
    username: foofoo
    password: barbar
`
	c := loadYaml(t, data)

	_, err := json.Marshal(c.Spec.Hosts[0].DaemonConfig)
	require.NoError(t, err)
}

func TestMigrateFromV1Beta1(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: mke
spec:
  engine:
    installURL: http://example.com/
  hosts:
  - address: "1.2.3.4"
    sshPort: 9022
    sshKeyPath: /path/to/nonexisting
    user: foofoo
    role: manager
  mke:
    username: foofoo
    password: barbar
`
	c := loadAndMigrateYaml(t, data)
	err := c.Validate()
	validateErrorField(t, err, "KeyPath")
	require.Equal(t, "launchpad.mirantis.com/mke/v1.2", c.APIVersion)

	require.Equal(t, c.Spec.MCR.InstallURLLinux, "http://example.com/")
	require.Equal(t, c.Spec.Hosts[0].SSH.Port, 9022)
	require.Equal(t, c.Spec.Hosts[0].SSH.User, "foofoo")
}

func TestMigrateFromV1Beta2(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta2
kind: mke
spec:
  engine:
    installURL: http://example.com/
  hosts:
  - address: "1.2.3.4"
    role: manager
    winRM:
      user: foo
      password: foo
  mke:
    username: foofoo
    password: barbar
`
	c := loadAndMigrateYaml(t, data)
	require.NoError(t, c.Validate())
	require.Equal(t, "launchpad.mirantis.com/mke/v1.2", c.APIVersion)
}

func TestMigrateFromV1Beta1WithoutInstallURL(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: mke
spec:
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
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
	require.Equal(t, "launchpad.mirantis.com/mke/v1.2", c.APIVersion)

	require.Equal(t, constant.MCRInstallURLLinux, c.Spec.MCR.InstallURLLinux)
	require.Equal(t, 9022, c.Spec.Hosts[0].SSH.Port)
	require.Equal(t, "foofoo", c.Spec.Hosts[0].SSH.User)
}

func TestHostWinRMCACertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: manager
      winRM:
        caCertPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "CACertPath")
}

func TestHostWinRMCertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: manager
      winRM:
        certPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := c.Validate()
	require.Error(t, err)
	validateErrorField(t, err, "CertPath")
}

func TestHostWinRMKeyPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: manager
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
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: manager
`
	c := loadYaml(t, data)

	require.Equal(t, c.Spec.Hosts[0].SSH.User, "root")
	require.Equal(t, c.Spec.Hosts[0].SSH.Port, 22)
	require.Equal(t, c.Spec.Hosts[0].SSH.KeyPath, "~/.ssh/id_rsa")
}

func TestHostWinRMDefaults(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: manager
      winRM:
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

func TestValidationWithMSRRole(t *testing.T) {

	t.Run("the role is not ucp, worker or msr", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
    - address: "1.2.3.4"
      role: weirdrole
    - address: "1.2.3.5"
      role: manager
`
		c := loadYaml(t, data)
		require.Error(t, c.Validate())
	})

	t.Run("the role is msr", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke+msr
spec:
  hosts:
    - address: "1.2.3.4"
      role: msr
      winRM:
        user: User
    - address: "1.2.3.5"
      role: manager
      winRM:
        user: User
`
		c := loadYaml(t, data)
		require.NoError(t, c.Validate())
	})

}

// Just a small helper to load the config struct from yaml to get defaults etc. in place
func loadYaml(t *testing.T, data string) *ClusterConfig {
	c := &ClusterConfig{}
	// convert any tabs added by editor into double spaces
	require.NoError(t, yaml.Unmarshal([]byte(strings.ReplaceAll(data, "\t", "  ")), c))
	return c
}

// Just a small helper to load the config struct from yaml through the migrations
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

// checks that the validation errors contains error for the expected field
func validateErrorField(t *testing.T, err error, field string) {
	fields := getAllErrorFields(err)
	require.Contains(t, fields, field)
}

func getAllErrorFields(err error) []string {
	validationErrors := err.(validator.ValidationErrors)
	fields := make([]string, len(validationErrors))

	// Collect all fields that failed validation
	// Also "store" the validation error for the expected field so that we can return it
	// and the correcponding test can further validate it if needed
	for _, fieldError := range validationErrors {
		fields = append(fields, fieldError.Field())
	}

	return fields
}
