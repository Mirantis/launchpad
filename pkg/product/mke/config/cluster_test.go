package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/config/migration"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v11"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v12"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v13"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v14"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v15"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1beta1"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1beta2"
	// needed to load the migrators.
	_ "github.com/Mirantis/launchpad/pkg/config/migration/v1beta3"
	"github.com/Mirantis/launchpad/pkg/constant"
	validator "github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
	mke:
	  version: 3.3.7
  hosts:
`
	c := loadYaml(t, data)
	err := c.Validate()
	require.Error(t, err)

	validateErrorField(t, err, "Hosts")
}

func TestMissingMCRChannelFails(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mke:
    version: 3.3.7
  hosts:
    - ssh:
        address: "1.2.3.4"
      role: manager
`
	c := &ClusterConfig{}
	err := yaml.Unmarshal([]byte(strings.ReplaceAll(data, "\t", "  ")), c)
	require.Error(t, err)
	require.ErrorContains(t, err, "spec.mcr.channel")
}

func TestHostAddressValidationWithInvalidIP(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
	data := `
apiVersion: launchpad.mirantis.com/v1beta1
kind: mke
spec:
	ucp:
	  version: 3.3.7
  engine:
    installURLWindows: http://example.com/install.ps1
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

	require.Equal(t, c.Spec.MCR.InstallURLWindows, "http://example.com/install.ps1")
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
    installURLWindows: http://example.com/install.ps1
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

	require.Equal(t, constant.MCRInstallURLWindows, c.Spec.MCR.InstallURLWindows)
	require.Equal(t, 9022, c.Spec.Hosts[0].SSH.Port)
	require.Equal(t, "foofoo", c.Spec.Hosts[0].SSH.User)
}

func TestHostWinRMCACertPathValidation(t *testing.T) {
	data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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
  mcr:
    channel: stable
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

func TestValidationWithMSRRole(t *testing.T) {
	kf, _ := os.CreateTemp("", "testkey")
	defer kf.Close()
	t.Run("the role is not ucp, worker or msr", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  mcr:
    channel: stable
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

	t.Run("the role is msr", func(t *testing.T) {
		data := `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke+msr
spec:
  mcr:
    channel: stable
	mke:
	  version: 3.3.7
	msr:
	  version: 2.8.5
  hosts:
    - ssh:
        address: "10.0.0.1"
			keyPath: ` + kf.Name() + `
      role: msr
    - ssh:
        address: "10.0.0.2"
			keyPath: ` + kf.Name() + `
      role: manager
`
		c := loadYaml(t, data)
		require.NoError(t, c.Validate())
	})

}

func TestWinRMPasswordExposureWarning(t *testing.T) {
	// An obviously fake value: the point of the assertions below is that this
	// string never reaches the log, so it must be recognisable if it ever does.
	const fakePassword = "NOT-A-REAL-PASSWORD"

	// Markers that tell the two warnings apart. A reader must be able to see
	// which failure applies, so the tests assert on the distinguishing phrase
	// rather than on "a warning happened".
	const (
		cleartextMarker = "sent in clear"
		unverifiedCert  = "winRM.caCertPath"
	)

	configWithWinRM := func(settings string) string {
		return `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mcr:
    channel: stable
  mke:
    version: 3.3.7
  hosts:
    - role: manager
      winRM:
        address: 10.0.0.1
        user: Administrator
` + settings
	}

	tests := []struct {
		name     string
		settings string
		// want is the phrase the single warning must contain, or "" when no
		// warning may be emitted at all.
		want string
	}{
		{
			name:     "insecure https with a password warns about the certificate",
			settings: "        useHTTPS: true\n        insecure: true\n        password: " + fakePassword + "\n",
			want:     unverifiedCert,
		},
		{
			// port 5986 implies TLS without an explicit useHTTPS: rig's
			// winrm.Config.SetDefaults flips UseHTTPS for that port, and
			// creasty/defaults invokes it from Host.UnmarshalYAML, so the
			// flag is already true by the time Validate runs. This case
			// locks that in: the warning must not depend on the user
			// having spelled useHTTPS out.
			name:     "insecure with implicit https via port 5986 warns about the certificate",
			settings: "        port: 5986\n        insecure: true\n        password: " + fakePassword + "\n",
			want:     unverifiedCert,
		},
		{
			// Added by user ruling, beyond the ticket's two named findings.
			// rig installs no transport decorator for plain HTTP without
			// NTLM, so go-winrm falls back to SetBasicAuth and the password
			// crosses the network with no encryption.
			name:     "plain http with a password warns about cleartext",
			settings: "        useHTTPS: false\n        password: " + fakePassword + "\n",
			want:     cleartextMarker,
		},
		{
			name:     "plain http with a password and insecure warns about cleartext, not the certificate",
			settings: "        useHTTPS: false\n        insecure: true\n        password: " + fakePassword + "\n",
			want:     cleartextMarker,
		},
		{
			// NTLM negotiates its own message-level encryption, so the
			// password is not cleartext even over plain HTTP.
			name:     "plain http with ntlm does not warn",
			settings: "        useHTTPS: false\n        useNTLM: true\n        password: " + fakePassword + "\n",
			want:     "",
		},
		{
			name:     "insecure https without a password does not warn",
			settings: "        useHTTPS: true\n        insecure: true\n",
			want:     "",
		},
		{
			name:     "verified https with a password does not warn",
			settings: "        useHTTPS: true\n        insecure: false\n        password: " + fakePassword + "\n",
			want:     "",
		},
		{
			name:     "host with no password does not warn",
			settings: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logrustest.NewGlobal()
			defer hook.Reset()

			c := loadYaml(t, configWithWinRM(tt.settings))
			require.NoError(t, c.Validate())

			var warnings []string
			for _, entry := range hook.AllEntries() {
				require.NotContains(t, entry.Message, fakePassword, "the warning must never print the password")
				// Structured fields are a separate leak route from the
				// formatted message: a change that moved the value into a
				// logrus field would otherwise go unnoticed here.
				require.NotContains(t, fmt.Sprint(entry.Data), fakePassword, "the password must never reach a log field")
				if entry.Level == logrus.WarnLevel {
					warnings = append(warnings, entry.Message)
				}
			}

			if tt.want == "" {
				require.Empty(t, warnings)
				return
			}

			require.Len(t, warnings, 1)
			require.Contains(t, warnings[0], "10.0.0.1")
			require.Contains(t, warnings[0], tt.want)

			// The two warnings must stay distinguishable: a host is told
			// either that its password crosses the wire unencrypted, or that
			// its peer is unauthenticated, never a blend of the two.
			other := cleartextMarker
			if tt.want == cleartextMarker {
				other = unverifiedCert
			}
			require.NotContains(t, warnings[0], other)
		})
	}
}

func TestWinRMPasswordExposureWarningIsPerHost(t *testing.T) {
	hook := logrustest.NewGlobal()
	defer hook.Reset()

	data := `
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mcr:
    channel: stable
  mke:
    version: 3.3.7
  hosts:
    - role: manager
      winRM:
        address: 10.0.0.1
        user: Administrator
        useHTTPS: true
        insecure: true
        password: NOT-A-REAL-PASSWORD
    - role: worker
      winRM:
        address: 10.0.0.2
        user: Administrator
        useHTTPS: false
        password: ALSO-NOT-REAL
    - role: worker
      winRM:
        address: 10.0.0.3
        user: Administrator
`
	c := loadYaml(t, data)
	require.NoError(t, c.Validate())

	var warnings []string
	for _, entry := range hook.AllEntries() {
		if entry.Level == logrus.WarnLevel {
			warnings = append(warnings, entry.Message)
		}
	}

	// One warning per affected host, each diagnosing that host's own
	// transport: 10.0.0.1 has an unverified certificate, 10.0.0.2 sends the
	// password in clear, and 10.0.0.3 has no password so is not reported.
	require.Len(t, warnings, 2)
	require.Contains(t, warnings[0], "10.0.0.1")
	require.Contains(t, warnings[0], "winRM.caCertPath")
	require.Contains(t, warnings[1], "10.0.0.2")
	require.Contains(t, warnings[1], "sent in clear")
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
