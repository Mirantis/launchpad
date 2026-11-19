// Package shellword decodes a single shell word the way a POSIX shell would.
//
// Launchpad's flag lists are written by hand in a config file, joined with
// spaces and then handed either to a remote shell or, since the credentials
// were taken out of argv, straight to a bootstrapper as an environment value.
// Both consumers have to read a quoted value identically, or the same config
// yields two different credentials depending on which code path uses it:
// --ucp-password="a\b" staged as a\b for the MSR install and as a\x08 for the
// MSR destroy, because the latter went through strconv.Unquote.
//
// Go string rules are the wrong rules here. strconv.Unquote reads "C:\temp"
// as C:<TAB>emp and does not decode a single-quoted word at all, neither of
// which is what the shell that used to receive these values did.
// rig's shellescape.Unquote is also not a POSIX decoder: inside double quotes
// it treats a backslash as escaping whatever follows, so "C:\temp" loses the
// backslash entirely.
package shellword

import "strings"

// Unquote decodes one shell word, without performing any expansion. A value
// may be single quoted, double quoted, backslash escaped or a concatenation
// of those: --registry-password="pa\$word" reaches the registry as pa$word.
//
// Expansions are deliberately not performed: $(...) and $VAR were expanded by
// the shell before these values were taken off the command line, and
// reproducing that here would execute config-supplied text inside launchpad.
// They are left as written, which is the safe difference.
//
// One deviation from a real shell: a word ending in an unescaped backslash
// escapes the following space when the flag list is joined, so the shell
// would merge the next flag into this value. That is not reachable from a
// shellescape.Quote'd value, and the alternative -- the merged word going to
// the bootstrapper as one argument -- is worse.
func Unquote(value string) string {
	var out strings.Builder

	for pos := 0; pos < len(value); {
		switch char := value[pos]; char {
		case '\'':
			end := strings.IndexByte(value[pos+1:], '\'')
			if end < 0 {
				out.WriteString(value[pos+1:])

				return out.String()
			}

			out.WriteString(value[pos+1 : pos+1+end])
			pos += end + 2
		case '"':
			consumed := writeDoubleQuoted(&out, value[pos+1:])
			pos += consumed + 1
		case '\\':
			if pos+1 < len(value) {
				out.WriteByte(value[pos+1])
				pos += 2

				continue
			}

			pos++
		default:
			out.WriteByte(char)
			pos++
		}
	}

	return out.String()
}

// writeDoubleQuoted writes the contents of a double-quoted section and
// returns how many bytes of it were consumed, including the closing quote
// when there is one. Inside double quotes a backslash escapes only ", \, $
// and a backtick; anywhere else it is literal.
func writeDoubleQuoted(out *strings.Builder, value string) int {
	for pos := 0; pos < len(value); pos++ {
		switch value[pos] {
		case '"':
			return pos + 1
		case '\\':
			if pos+1 < len(value) && strings.IndexByte(`"\$`+"`", value[pos+1]) >= 0 {
				out.WriteByte(value[pos+1])
				pos++

				continue
			}

			out.WriteByte('\\')
		default:
			out.WriteByte(value[pos])
		}
	}

	return len(value)
}
