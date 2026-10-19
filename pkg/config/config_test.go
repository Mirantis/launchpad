package config

import (
	"os"
	"testing"

	"github.com/Mirantis/launchpad/pkg/product/mke"
	"github.com/stretchr/testify/require"
)

// baseConfig returns a minimal, otherwise-valid MKE cluster config with the
// given adminPassword substituted in, so ProductFromYAML's envsubst pass is
// exercised end to end (not just at the raw envsubst.Bytes layer) and, when
// substitution succeeds, the config actually parses into a Product.
func baseConfig(adminPassword string) []byte {
	return []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      user: root
  mcr:
    channel: stable-29.4
  mke:
    adminPassword: "` + adminPassword + `"
    version: "3.7.3"
`)
}

func TestProductFromYAMLRejectsUnsetDollarReferenceInPassword(t *testing.T) {
	// A generated password containing an unescaped "$word" that doesn't
	// correspond to a set environment variable must not be silently
	// truncated at the "$" -- it must fail loudly so the caller can see
	// their credential would otherwise be corrupted.
	_, err := ProductFromYAML(baseConfig(`Qr0!@dGo7ukgWC0$yAzL`))
	require.ErrorContains(t, err, "failed to substitute environment variables")
	require.ErrorContains(t, err, "variable ${yAzL} not set")
}

func TestProductFromYAMLPreservesDigitLeadingDollarInPassword(t *testing.T) {
	// "$" followed by a digit can never be a valid environment variable
	// name, so it must be preserved as literal text rather than rejected
	// or substituted.
	p, err := ProductFromYAML(baseConfig(`hasdigit$5986after`))
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestProductFromYAMLPreservesPlainPassword(t *testing.T) {
	// Passwords with no "$" at all must be entirely unaffected.
	p, err := ProductFromYAML(baseConfig(`Mn0&33IYpG55NqDYmB3J`))
	require.NoError(t, err)
	require.NotNil(t, p)
}

func TestProductFromYAMLStillSubstitutesSetEnvironmentVariables(t *testing.T) {
	// Legitimate use of the feature -- referencing a set environment
	// variable -- must keep working, and must resolve to the actual value.
	t.Setenv("LAUNCHPAD_TEST_MKE_PASSWORD", "s3cr3t")
	p, err := ProductFromYAML(baseConfig(`${LAUNCHPAD_TEST_MKE_PASSWORD}`))
	require.NoError(t, err)
	m, ok := p.(*mke.MKE)
	require.True(t, ok)
	require.Equal(t, "s3cr3t", m.ClusterConfig.Spec.MKE.AdminPassword)
}

func TestProductFromYAMLAllowsEscapedDollar(t *testing.T) {
	// A literal "$" can still be produced by escaping it as "$$", per
	// envsubst's own escaping convention, and must resolve to the intended
	// literal password rather than some other value.
	require.Empty(t, os.Getenv("yAzL"))
	p, err := ProductFromYAML(baseConfig(`Qr0!@dGo7ukgWC0$$yAzL`))
	require.NoError(t, err)
	m, ok := p.(*mke.MKE)
	require.True(t, ok)
	require.Equal(t, `Qr0!@dGo7ukgWC0$yAzL`, m.ClusterConfig.Spec.MKE.AdminPassword)
}

// The migrators register themselves from init(), so these tests go through
// ProductFromYAML without importing any migrator directly: a migrator that
// pkg/config does not import leaves old configs unmigrated and fails here.
func TestProductFromYAMLMigratesV15Config(t *testing.T) {
	p, err := ProductFromYAML([]byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      user: root
  mcr:
    channel: stable
    version: 25.0.14
  mke:
    version: "3.8.6"
`))
	require.NoError(t, err)
	m, ok := p.(*mke.MKE)
	require.True(t, ok)
	require.Equal(t, "launchpad.mirantis.com/mke/v1.6", m.ClusterConfig.APIVersion)
	require.Equal(t, "stable-25.0.14", m.ClusterConfig.Spec.MCR.Channel)
}

func TestProductFromYAMLMigratesV14Config(t *testing.T) {
	p, err := ProductFromYAML([]byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      user: root
  mcr:
    channel: stable
    version: "25.0"
  mke:
    version: "3.8.6"
    swarmInstallFlags:
    - --default-addr-pool=172.20.0.0/14
`))
	require.NoError(t, err)
	m, ok := p.(*mke.MKE)
	require.True(t, ok)
	require.Equal(t, "launchpad.mirantis.com/mke/v1.6", m.ClusterConfig.APIVersion)
	require.Equal(t, "stable-25.0", m.ClusterConfig.Spec.MCR.Channel)
	require.Equal(t, []string{"--default-addr-pool=172.20.0.0/14"}, []string(m.ClusterConfig.Spec.MCR.SwarmInstallFlags))
}
