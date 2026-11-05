package phase

import (
	"strings"
	"testing"

	rigcmd "github.com/k0sproject/rig/v2/cmd"
	"github.com/k0sproject/rig/v2/sh/shellescape"
	"github.com/stretchr/testify/require"
)

// redact applies the values the way rig does: cmd.Redact appends to a list
// of literal matches, and each is removed with strings.ReplaceAll
// (cmd/execoptions.go). The harness previously compiled each value as a
// regular expression, which is what the production code assumed when it
// wrapped them in regexp.QuoteMeta -- so both sides agreed with each other
// and neither agreed with rig.
func redact(t *testing.T, values []string, rendered string) string {
	t.Helper()

	opts := rigcmd.Build(values2options(values)...)

	return opts.RedactString(rendered)
}

func values2options(values []string) []rigcmd.ExecOption {
	opts := make([]rigcmd.ExecOption, 0, len(values))
	for _, value := range values {
		opts = append(opts, rigcmd.Redact(value))
	}

	return opts
}

// TestAdminCredentialRedactionsCoverRenderedFlags pins the contract between the
// flags as rendered on the command line and the patterns that hide them: the
// credentials are shell-quoted into the command, so redaction must match the
// quoted form too.
func TestAdminCredentialRedactionsCoverRenderedFlags(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		username string
		password string
	}{
		{name: "no quoting needed", username: "admin", password: "orcaorca"},
		{name: "single quote in password", username: "admin", password: `p@ss'w0rd`},
		{name: "single quote in username", username: `ad'min`, password: "orcaorca"},
		{name: "space and metacharacters", username: "admin user", password: "$(id); rm -rf /"},
		{name: "username only", username: `ad'min`, password: ""},
		{name: "password only", username: "", password: `p@ss'w0rd`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rendered := strings.Join(adminCredentialFlags(tc.username, tc.password), " ")
			redacted := redact(t, adminCredentialRedactions(tc.username, tc.password), rendered)

			if tc.username != "" {
				require.NotContains(t, redacted, tc.username, "username survived redaction of %q", rendered)
				require.NotContains(t, redacted, shellescape.Quote(tc.username), "quoted username survived redaction of %q", rendered)
			}
			if tc.password != "" {
				require.NotContains(t, redacted, tc.password, "password survived redaction of %q", rendered)
				require.NotContains(t, redacted, shellescape.Quote(tc.password), "quoted password survived redaction of %q", rendered)
			}
		})
	}
}

// TestAdminCredentialRedactionsMatchQuotedForm is the regression for the exact
// leak: shell-quoting a value containing a single quote rewrites it, so a
// pattern built only from the raw value no longer matches what is logged.
func TestAdminCredentialRedactionsMatchQuotedForm(t *testing.T) {
	t.Parallel()

	const password = `p@ss'w0rd`

	rendered := strings.Join(adminCredentialFlags("admin", password), " ")
	// The rendered flag really is the rewritten form, not the raw value.
	// shellescape closes the quote, emits an escaped quote, and reopens it.
	require.Contains(t, rendered, `--admin-password 'p@ss'"'"'w0rd'`)

	require.NotContains(t, rendered, password,
		"the raw value unexpectedly appears in %q; the regression this guards would not be detectable", rendered)

	require.NotContains(t, redact(t, adminCredentialRedactions("admin", password), rendered), "w0rd")
}

// TestAdminCredentialRedactionsSkipEmptyValues keeps an empty credential from
// producing an empty pattern, which would match at every position and redact
// the entire installer output.
func TestAdminCredentialRedactionsSkipEmptyValues(t *testing.T) {
	t.Parallel()

	require.Empty(t, adminCredentialRedactions("", ""))
	for _, pattern := range adminCredentialRedactions("admin", "") {
		require.NotEmpty(t, pattern)
	}
}
