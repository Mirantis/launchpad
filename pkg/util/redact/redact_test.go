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
	for _, tt := range []struct {
		name string
		flag string
		want string
	}{
		{"equals form", "--admin-password=secret", "--admin-password=" + redact.Placeholder},
		{"quoted equals form", "--admin-password='secret'", "--admin-password=" + redact.Placeholder},
		{"space form", "--ucp-password secret", "--ucp-password " + redact.Placeholder},
		{"username flag", "--ucp-username=secret", "--ucp-username=" + redact.Placeholder},
		{"unknown flag untouched", "--san=mke.example.com", "--san=mke.example.com"},
		{"valueless flag untouched", "--force-minimums", "--force-minimums"},
		{"bare sensitive flag untouched", "--admin-password", "--admin-password"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := redact.YAML([]byte("installFlags:\n- " + tt.flag + "\n"))
			require.NoError(t, err)

			var parsed struct {
				InstallFlags []string `yaml:"installFlags"`
			}
			require.NoError(t, yaml.Unmarshal(out, &parsed))
			require.Equal(t, []string{tt.want}, parsed.InstallFlags)
		})
	}
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
