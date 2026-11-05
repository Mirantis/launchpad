package phase

import (
	"regexp"
	"strings"
	"testing"

	"github.com/k0sproject/rig/v2/sh/shellescape"
	"github.com/stretchr/testify/require"
)

// redact applies the patterns the way cmd.Redact does, so the test exercises
// the rendered command rather than the pattern strings in isolation.
func redact(t *testing.T, patterns []string, rendered string) string {
	t.Helper()

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		require.NoError(t, err, "pattern %q does not compile", pattern)
		rendered = re.ReplaceAllString(rendered, "[REDACTED]")
	}

	return rendered
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

	rawOnly := regexp.MustCompile(regexp.QuoteMeta(password))
	require.False(t, rawOnly.MatchString(rendered),
		"raw-value-only pattern unexpectedly matches %q; the regression this guards would not be detectable", rendered)

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
