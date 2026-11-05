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
// masking it gets is the old regex, which is best-effort: it is case
// sensitive and knows only two key names.
func TestRedactConfigDumpFallsBackOnUnparseableInput(t *testing.T) {
	dump := redactConfigDump([]byte("spec:\n  hosts:\n - broken\n\tpassword: fake-host-password\n"))

	require.NotContains(t, dump, "fake-host-password", "the fallback must still mask what it can match")
	require.Contains(t, dump, "[REDACTED]")
}

// The fallback's weakness is the reason it is documented rather than
// trusted: it matches lowercase username and password only, so a camelCase
// key in an unparseable document is not masked. This pins the limit so that
// nobody reads the fallback as equivalent to the parsed path.
func TestRedactConfigDumpFallbackDoesNotCoverSchemaKeys(t *testing.T) {
	dump := redactConfigDump([]byte("spec:\n  mke:\n - broken\n\tadminPassword: fake-admin-password\n"))

	require.Contains(t, dump, "fake-admin-password",
		"if this now masks, the fallback improved and its documented limit should be updated")
}
