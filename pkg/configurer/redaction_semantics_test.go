package configurer_test

import (
	"regexp"
	"testing"

	rigcmd "github.com/k0sproject/rig/v2/cmd"
	"github.com/stretchr/testify/require"
)

// TestRigRedactionIsLiteral pins how rig treats the values passed to
// cmd.Redact, at the version go.mod resolves to.
//
// Launchpad's redaction sites depend on two properties that are not visible
// from the call site and that both plausible readings of the API get wrong in
// opposite directions:
//
//   - the values are matched literally, not as regular expressions, so
//     wrapping a credential in regexp.QuoteMeta produces a spelling that
//     never appears in the output. That failure is invisible in testing,
//     because QuoteMeta is the identity function for alphanumerics: only a
//     credential containing a regex metacharacter exposes it.
//   - repeated cmd.Redact options compose rather than overwrite, so a call
//     site may list the raw and the shell-quoted form of the same value.
//
// Neither property is safe to assume from reading the API: rig's v1 Redact
// compiled a regexp, so the opposite was once true. This branch was written
// against a fork of rig and these tests passed there; they pass unchanged
// against upstream v2.2.1, which launchpad now builds on (#707). If a future
// bump changes the semantics, this fails rather than the redaction silently
// doing nothing.
func TestRigRedactionIsLiteral(t *testing.T) {
	t.Parallel()

	const password = `p$ss.w*rd`

	opts := rigcmd.Build(rigcmd.Redact(password))
	require.Equal(t, "login -p "+rigcmd.DefaultRedactMask, opts.RedactString("login -p "+password),
		"a credential containing regex metacharacters must be matched literally")

	// The same value escaped for a regex matcher is not what appears in the
	// command, so it redacts nothing. This is the defect the production
	// sites had.
	escaped := rigcmd.Build(rigcmd.Redact(regexp.QuoteMeta(password)))
	require.Equal(t, "login -p "+password, escaped.RedactString("login -p "+password),
		"regexp.QuoteMeta must not be applied to a value handed to cmd.Redact")
}

func TestRigRedactionsCompose(t *testing.T) {
	t.Parallel()

	opts := rigcmd.Build(rigcmd.Redact("alice"), rigcmd.Redact("hunter2"))

	got := opts.RedactString("login -u alice -p hunter2")

	require.NotContains(t, got, "alice", "an earlier cmd.Redact must not be discarded by a later one")
	require.NotContains(t, got, "hunter2")
}
