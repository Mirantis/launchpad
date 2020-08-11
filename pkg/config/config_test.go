package config

import (
	"encoding/json"
	"strings"
	"testing"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	"github.com/Mirantis/mcc/pkg/constant"
	validator "github.com/go-playground/validator/v10"

	"github.com/stretchr/testify/require"
)

func TestNonExistingHostsFails(t *testing.T) {
	data := `
apiVersion: "launchpad.mirantis.com/v1beta3"
kind: DockerEnterprise
spec:
  hosts:
`
	c := loadYaml(t, data)
	err := Validate(c)
	require.Error(t, err)

	validateErrorField(t, err, "Hosts")
}

func TestHostAddressValidationWithInvalidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "512.1.2.3"
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "Address")
}

func TestHostAddressValidationWithValidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.10.10.10"
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.NotContains(t, getAllErrorFields(err), "Address")
}

func TestHostAddressValidationWithInvalidHostname(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1-2-foo"
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "Address")
}

func TestHostAddressValidationWithValidHostname(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "foo.example.com"
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.NotContains(t, getAllErrorFields(err), "Address")

}

func TestHostSshPortValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: manager
			ssh:
        port: 0
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "Port")
}

func TestHostSshKeyValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
      ssh:
        port: 22
        keyPath: /path/to/nonexisting/key
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
}

func TestHostRoleValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
  - address: "1.2.3.4"
    ssh:
		  port: 22
    role: foobar
`
	c := loadYaml(t, data)
	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "Role")
}

func TestHostWithComplexEngineConfig(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
  - address: "1.2.3.4"
		ssh:
		  port: 22
    role: worker
    engineConfig:
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
	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: UCP
spec:
  engine:
	  installURL: http://example.com/
  hosts:
  - address: "1.2.3.4"
		sshPort: 9022
		sshKeyPath: /path/to/nonexisting
		user: foofoo
    role: manager
`
	c := loadYaml(t, data)
	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
	require.Equal(t, c.APIVersion, "launchpad.mirantis.com/v1beta3")

	require.Equal(t, c.Spec.Engine.InstallURLLinux, "http://example.com/")
	require.Equal(t, c.Spec.Hosts[0].SSH.Port, 9022)
	require.Equal(t, c.Spec.Hosts[0].SSH.User, "foofoo")
}

func TestMigrateFromV1Beta2(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta2
kind: UCP
spec:
  engine:
	  installURL: http://example.com/
  hosts:
  - address: "1.2.3.4"
    role: manager
		winRM:
		  user: foo
			password: foo
`
	c := loadYaml(t, data)
	require.NoError(t, Validate(c))
	require.Equal(t, c.APIVersion, "launchpad.mirantis.com/v1beta3")
}

func TestMigrateFromV1Beta1WithoutInstallURL(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: UCP
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
	c := loadYaml(t, data)
	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
	require.Equal(t, c.APIVersion, "launchpad.mirantis.com/v1beta3")

	require.Equal(t, c.Spec.Engine.InstallURLLinux, constant.EngineInstallURLLinux)
	require.Equal(t, c.Spec.Hosts[0].SSH.Port, 9022)
	require.Equal(t, c.Spec.Hosts[0].SSH.User, "foofoo")
}

func TestHostWinRMCACertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: manager
		  winRM:
			  caCertPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "CACertPath")
}

func TestHostWinRMCertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: manager
		  winRM:
			  certPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "CertPath")
}

func TestHostWinRMKeyPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: manager
		  winRM:
			  keyPath: /path/to/nonexisting
`
	c := loadYaml(t, data)

	err := Validate(c)
	require.Error(t, err)
	validateErrorField(t, err, "KeyPath")
}

func TestHostSSHDefaults(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
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
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: manager
		  winRM:
			  user: User
`
	c := loadYaml(t, data)

	require.NoError(t, Validate(c))

	require.Equal(t, c.Spec.Hosts[0].WinRM.User, "User")
	require.Equal(t, c.Spec.Hosts[0].WinRM.Port, 5985)
	require.Equal(t, c.Spec.Hosts[0].WinRM.UseNTLM, false)
	require.Equal(t, c.Spec.Hosts[0].WinRM.UseHTTPS, false)
	require.Equal(t, c.Spec.Hosts[0].WinRM.Insecure, false)
}

func TestValidationWithDtrRole(t *testing.T) {

	t.Run("the role is not ucp, worker or dtr", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: weirdrole
`
		c := loadYaml(t, data)

		require.Error(t, Validate(c))
	})

	t.Run("the role is dtr", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
    - address: "1.2.3.4"
		  role: dtr
		  winRM:
		    user: User
`
		c := loadYaml(t, data)

		require.NoError(t, Validate(c))
	})

}

// Just a small helper to load the config struct from yaml to get defaults etc. in place
func loadYaml(t *testing.T, data string) *api.ClusterConfig {
	// convert any tabs added by editor into double spaces
	c, err := FromYaml([]byte(strings.ReplaceAll(data, "\t", "  ")))
	if err != nil {
		t.Error(err)
	}
	return &c
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
