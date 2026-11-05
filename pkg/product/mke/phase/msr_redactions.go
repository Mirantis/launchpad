package phase

import (
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
)

// msrCredentialFlags are the MSR bootstrapper flags whose values are
// credentials and which launchpad puts on the command line.
var msrCredentialFlags = []string{"--ucp-username", "--ucp-password"}

// msrCredentialRedactions returns the literal strings to keep out of the log
// for an MSR bootstrapper invocation.
//
// Both the value as it appears in the flag list and the value with its
// surrounding quotes removed are returned: BuildMKEFlags writes
// --ucp-password="secret", so the text in the command is the quoted form,
// while an operator grepping the log for the configured password looks for
// the raw one. rig matches redactions literally, so each spelling has to be
// listed separately; an empty value is skipped, because an empty match
// would redact the entire output.
func msrCredentialRedactions(flags commonconfig.Flags) []string {
	var values []string

	for _, flag := range msrCredentialFlags {
		quoted := flags.Get(flag)
		if quoted == "" {
			continue
		}

		if value := commonconfig.FlagValue(quoted); value != "" {
			values = append(values, value)
		}

		if idx := indexOfValue(quoted); idx >= 0 && idx < len(quoted)-1 {
			values = append(values, quoted[idx+1:])
		}
	}

	return values
}

// indexOfValue returns the position of the separator between a flag and its
// value, or -1 when the flag carries none.
func indexOfValue(flag string) int {
	for i, char := range flag {
		if char == '=' || char == ' ' {
			return i
		}
	}

	return -1
}
