package redact_test

import (
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/util/redact"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestYAMLMasksCredentialKeys(t *testing.T) {
	doc := []byte(`
spec:
  hosts:
  - winRM:
      address: 10.0.0.1
      user: Administrator
      password: host-secret
    role: manager
  mke:
    adminUsername: admin-name
    adminPassword: mke-secret
    keyData: |
      -----BEGIN PRIVATE KEY-----
      key-secret
    caCertData: public-ca-material
  mcr:
    channel: stable-29.4
    license: license-secret
`)

	out, err := redact.YAML(doc)
	require.NoError(t, err)

	got := string(out)
	for _, secret := range []string{"host-secret", "mke-secret", "admin-name", "key-secret", "license-secret"} {
		require.NotContainsf(t, got, secret, "redacted output still contains %q:\n%s", secret, got)
	}

	require.Contains(t, got, "public-ca-material", "caCertData is public trust material and must survive")
	require.Contains(t, got, "Administrator", "the connection user is not a credential")
	require.Contains(t, got, "10.0.0.1")
	require.Contains(t, got, "stable-29.4")
	require.Equal(t, 5, strings.Count(got, redact.Placeholder))
}

func TestYAMLMasksFlagArguments(t *testing.T) {
	const mask = redact.Placeholder

	for _, tt := range []struct {
		name string
		flag string
		want string
	}{
		{"equals form", "--admin-password=secret", "--admin-password=" + mask},
		{"quoted equals form", "--admin-password='secret'", "--admin-password=" + mask},
		{"space form", "--ucp-password secret", "--ucp-password " + mask},
		{"username flag", "--ucp-username=secret", "--ucp-username=" + mask},
		{"unknown flag untouched", "--san=mke.example.com", "--san=mke.example.com"},
		{"valueless flag untouched", "--force-minimums", "--force-minimums"},
		{"bare sensitive flag untouched", "--admin-password", "--admin-password"},
		// Flags.Join() concatenates the list into one shell fragment, so
		// several flags in one element are legal and effective.
		{"grouped flags", "--ucp-insecure-tls --ucp-password=secret", "--ucp-insecure-tls --ucp-password=" + mask},
		{"grouped flags, credential first", "--admin-password=secret --debug", "--admin-password=" + mask + " --debug"},
		{"leading whitespace", " --admin-password=secret", " --admin-password=" + mask},
		{"tab separator", "--admin-password\tsecret", "--admin-password\t" + mask},
		{"repeated spacing preserved", "--san=a  --admin-password=b   --debug", "--san=a  --admin-password=" + mask + "   --debug"},
		{"quoted value containing spaces", "--admin-password='my secret'", "--admin-password=" + mask},
		{"quoted value in space form", "--admin-password 'my secret'", "--admin-password " + mask},
		{"value is not mistaken for a flag", "--san=--admin-password=x", "--san=--admin-password=x"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := yaml.Marshal(map[string][]string{"installFlags": {tt.flag}})
			require.NoError(t, err)

			out, err := redact.YAML(doc)
			require.NoError(t, err)

			var parsed struct {
				InstallFlags []string `yaml:"installFlags"`
			}
			require.NoError(t, yaml.Unmarshal(out, &parsed))
			require.Equal(t, []string{tt.want}, parsed.InstallFlags)
			require.NotContains(t, string(out), "secret")
		})
	}
}

// MKEConfig.UnmarshalYAML loads spec.mke.configFile and
// spec.mke.cloud.configFile into configData, so a credential can reach the
// dump even when the original YAML only referenced a file path.
func TestYAMLMasksConfigData(t *testing.T) {
	doc := []byte(`
spec:
  mke:
    configData: |
      [auth.ldap]
      reader_dn_password = "ldap-secret"
    cloud:
      provider: azure
      configData: |
        {"aadClientSecret": "cloud-secret"}
`)

	out, err := redact.YAML(doc)
	require.NoError(t, err)

	got := string(out)
	require.NotContains(t, got, "ldap-secret")
	require.NotContains(t, got, "cloud-secret")
	require.Contains(t, got, "provider: azure", "the provider name is not a credential")
}

// A flag and its value may be two separate list elements, in which case the
// value carries no flag name of its own.
func TestYAMLMasksTwoElementFlagArgument(t *testing.T) {
	out, err := redact.YAML([]byte("installFlags:\n- --admin-password\n- secret\n- --san\n- mke.example.com\n"))
	require.NoError(t, err)

	var parsed struct {
		InstallFlags []string `yaml:"installFlags"`
	}
	require.NoError(t, yaml.Unmarshal(out, &parsed))
	require.Equal(t, []string{"--admin-password", redact.Placeholder, "--san", "mke.example.com"}, parsed.InstallFlags)
}

func TestYAMLPreservesKeyOrderAndEmptyValues(t *testing.T) {
	out, err := redact.YAML([]byte("kind: mke\napiVersion: v1.6\nspec:\n  mke:\n    version: 3.7.0\n    adminPassword: \"\"\n"))
	require.NoError(t, err)

	require.Equal(t, "kind: mke\napiVersion: v1.6\nspec:\n  mke:\n    version: 3.7.0\n    adminPassword: \"\"\n", string(out))
}

func TestYAMLRejectsNonMappingDocument(t *testing.T) {
	_, err := redact.YAML([]byte("- just\n- a list\n"))
	require.Error(t, err)
}
