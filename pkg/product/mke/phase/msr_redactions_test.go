package phase

import (
	"testing"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	rigcmd "github.com/k0sproject/rig/v2/cmd"
	"github.com/stretchr/testify/require"
)

// mask applies the values the way rig does, through the real code path.
func mask(values []string, rendered string) string {
	opts := make([]rigcmd.ExecOption, 0, len(values))
	for _, value := range values {
		opts = append(opts, rigcmd.Redact(value))
	}

	return rigcmd.Build(opts...).RedactString(rendered)
}

// BuildMKEFlags shell-quotes the credential, so the command line holds the
// quoted spelling while the bootstrapper's own output may echo the raw one.
// Both have to be registered, because rig matches literally.
func TestMSRCredentialRedactionsCoverQuotedAndRawForms(t *testing.T) {
	const password = `p@ss'w0rd`

	flags := commonconfig.Flags{`--ucp-username 'admin'`, `--ucp-password 'p@ss'"'"'w0rd'`}

	values := msrCredentialRedactions(flags)

	require.NotContains(t, mask(values, "dtr install "+flags.Join()), "w0rd",
		"the rendered command still shows the password")
	require.NotContains(t, mask(values, "authentication failed for "+password), "w0rd",
		"the raw password still shows where the bootstrapper echoes it")
}

// Flags.Join concatenates the list, so a flag and its value in separate
// elements form a working command. Flags.Get returns only the flag, so the
// value has to be read from the next element or it is never redacted.
func TestMSRCredentialRedactionsCoverTwoElementForm(t *testing.T) {
	flags := commonconfig.Flags{"--ucp-password", "split-secret", "--ucp-node", "host-1"}

	values := msrCredentialRedactions(flags)

	require.NotContains(t, mask(values, "dtr upgrade "+flags.Join()), "split-secret")
}

// rig replaces each registered value in turn, so a username that is a
// prefix of the password must not be substituted first: doing so destroys
// the longer match and leaves the rest of the password in the log.
func TestMSRCredentialRedactionsOrderLongestFirst(t *testing.T) {
	flags := commonconfig.Flags{"--ucp-username admin", "--ucp-password admin!s3cr3t"}

	values := msrCredentialRedactions(flags)

	require.NotContains(t, mask(values, "dtr install "+flags.Join()), "s3cr3t")
}

// A flag that carries no value registers nothing: an empty redaction would
// match at every position and blank the whole output.
func TestMSRCredentialRedactionsIgnoreValuelessFlags(t *testing.T) {
	require.Empty(t, msrCredentialRedactions(commonconfig.Flags{"--ucp-password", "--ucp-node host-1"}))
	require.Empty(t, msrCredentialRedactions(commonconfig.Flags{"--ucp-password="}))
	require.Empty(t, msrCredentialRedactions(commonconfig.Flags{"--ucp-node host-1"}))
}
