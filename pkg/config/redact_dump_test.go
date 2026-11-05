package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The regex this replaced was case sensitive and matched neither the
// schema's camelCase keys nor the inline data fields, so a configuration
// dumped to the log carried the admin password in clear.
func TestRedactConfigDumpMasksSchemaKeys(t *testing.T) {
	dump := redactConfigDump([]byte(`apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  hosts:
  - winRM:
      address: 10.0.0.1
      password: fake-host-password
    role: manager
    environment:
      REGISTRY_TOKEN: fake-registry-token
  mke:
    adminUsername: fake-admin-user
    adminPassword: fake-admin-password
    keyData: fake-key-material
`))

	for _, secret := range []string{
		"fake-host-password",
		"fake-registry-token",
		"fake-admin-user",
		"fake-admin-password",
		"fake-key-material",
	} {
		require.NotContainsf(t, dump, secret, "the config dump leaked %q:\n%s", secret, dump)
	}

	require.Contains(t, dump, "10.0.0.1", "non-credential configuration must survive, or the dump is useless")
}

// A configuration that cannot be parsed is still dumped, because a debug
// dump of a broken configuration is exactly when the output is wanted. The
// masking it gets is line oriented rather than structural, but it covers
// the same key set, so camelCase schema keys and credential-named
// environment variables are masked there too.
func TestRedactConfigDumpFallsBackOnUnparseableInput(t *testing.T) {
	dump := redactConfigDump([]byte(
		"spec:\n  hosts:\n - broken\n\tpassword: fake-host-password\n" +
			"\tadminPassword: fake-admin-password\n" +
			"\tREGISTRY_TOKEN: fake-registry-token\n"))

	for _, secret := range []string{"fake-host-password", "fake-admin-password", "fake-registry-token"} {
		require.NotContainsf(t, dump, secret, "the fallback left %q in an unparseable config dump:\n%s", secret, dump)
	}
}

// The fallback's real limit, and the reason the log line calls it
// best-effort: it works a line at a time, so a credential in a block scalar
// is not masked. The parsed path has no such gap. This pins the boundary so
// that nobody reads the two as equivalent.
func TestRedactConfigDumpFallbackCannotSeeBlockScalars(t *testing.T) {
	dump := redactConfigDump([]byte(
		"spec:\n  mke:\n - broken\n\tkeyData: |\n\t  fake-key-line-one\n\t  fake-key-line-two\n"))

	require.Contains(t, dump, "fake-key-line-one",
		"if this now masks, the fallback gained multi-line coverage and the log line's wording should be revisited")
}
