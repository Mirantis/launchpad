package phase

import (
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/Mirantis/launchpad/pkg/util/redact"
	"github.com/k0sproject/rig/v2/sh/shellescape"
)

// msrCredentialFlags are the MSR bootstrapper flags whose values are
// credentials and which launchpad puts on the command line.
var msrCredentialFlags = []string{"--ucp-username", "--ucp-password"}

// msrCredentialRedactions returns the literal strings to keep out of the log
// for an MSR bootstrapper invocation, longest first.
//
// Three spellings of the same credential can reach the log and each has to
// be registered separately, because rig matches literally:
//
//   - the rendered form, which is what BuildMKEFlags wrote into the flag
//     list and what appears in the command line. BuildMKEFlags shell-quotes
//     it, so for a password containing a quote this is not the password.
//   - the shell-decoded form, which is the password itself. It reaches the
//     log if the bootstrapper echoes it back in its own output.
//   - the two-element form, where a flag and its value are separate entries
//     in the list. Flags.Join concatenates them into a working command, but
//     Flags.Get returns only the flag, so the value has to be read from the
//     following element or it is never registered at all.
//
// Ordering matters as much as coverage: rig replaces each value in turn, so
// a username that is a prefix of the password must not be substituted
// first. redact.ByLengthDesc removes that dependence on order.
func msrCredentialRedactions(flags commonconfig.Flags) []string {
	var values []string

	for _, name := range msrCredentialFlags {
		for _, rendered := range renderedFlagValues(flags, name) {
			values = append(values, rendered)

			if decoded, err := shellescape.Unquote(rendered); err == nil && decoded != rendered {
				values = append(values, decoded)
			}
		}
	}

	return redact.ByLengthDesc(values...)
}

// renderedFlagValues returns every value the named flag carries in the list,
// covering both "--flag value" or "--flag=value" in one element and a bare
// "--flag" whose value is the next element.
func renderedFlagValues(flags commonconfig.Flags, name string) []string {
	var rendered []string

	for i, flag := range flags {
		if flag == name {
			if i+1 < len(flags) && !isFlagToken(flags[i+1]) {
				rendered = append(rendered, flags[i+1])
			}

			continue
		}

		if !isFlagWithValue(flag, name) {
			continue
		}

		if value := flag[len(name)+1:]; value != "" {
			rendered = append(rendered, value)
		}
	}

	return rendered
}

func isFlagWithValue(flag, name string) bool {
	if len(flag) <= len(name) || flag[:len(name)] != name {
		return false
	}

	return flag[len(name)] == '=' || flag[len(name)] == ' '
}

func isFlagToken(value string) bool {
	return len(value) > 0 && value[0] == '-'
}
