package phase

import (
	"sort"

	"github.com/k0sproject/rig/v2/cmd"
)

// redactOptions builds the rig redaction options for a set of secret values.
//
// Registration order is load bearing and is not something a call site can be
// trusted to get right by hand. rig applies redactions sequentially in the
// order they were registered (cmd/execoptions.go RedactString, and
// redact.Writer, which keeps the first match that starts at a position), so a
// shorter value that is a prefix of a longer one is replaced first and
// rewrites the longer value's text before the longer value's own pass runs.
// The remainder then survives. The MKE admin credentials are exactly that
// shape in the default configuration: install_mke.go defaults the username to
// "admin", so a password beginning "admin" is masked only down to
// "[REDACTED]!s3cr3t". Sorting longest first makes the containing value win
// whatever order the caller lists its secrets in.
//
// Empty values are dropped. An empty match means "every position":
// ExecOptions.RedactString expands it into a mask between every character of
// the output, and a value that is not set is not a secret worth matching
// anyway. The admin username and password are both routinely unset.
//
// Duplicates are dropped so that a value supplied twice (a username that
// equals the password, say) does not register two identical passes.
func redactOptions(values ...string) []cmd.ExecOption {
	secrets := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, dup := seen[value]; dup {
			continue
		}
		seen[value] = struct{}{}
		secrets = append(secrets, value)
	}

	// Stable so that two secrets of equal length keep the caller's order,
	// which keeps the built options reproducible across runs.
	sort.SliceStable(secrets, func(i, j int) bool {
		return len(secrets[i]) > len(secrets[j])
	})

	opts := make([]cmd.ExecOption, 0, len(secrets))
	for _, secret := range secrets {
		opts = append(opts, cmd.Redact(secret))
	}

	return opts
}
