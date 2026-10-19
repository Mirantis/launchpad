package configurer

import (
	"strings"

	"github.com/k0sproject/rig/v2/sh/shellescape"
)

// DockerCmd builds a docker CLI command line for remote execution with every
// interpolated argument quoted by construction.
//
// Remote commands are handed to a shell on the target host (a login shell on
// Linux, cmd.exe via WinRM on Windows), so any value that reaches the command
// line unquoted is executable. Build commands with Arg/Args and reserve Raw
// for values that are known constants or are already quoted.
type DockerCmd struct {
	quote func(string) string
	parts []string
}

// Arg appends a single argument, quoted for the target shell.
func (d *DockerCmd) Arg(arg string) *DockerCmd {
	d.parts = append(d.parts, d.quote(arg))

	return d
}

// Args appends multiple arguments, each quoted for the target shell.
func (d *DockerCmd) Args(args ...string) *DockerCmd {
	for _, arg := range args {
		d.Arg(arg)
	}

	return d
}

// Raw appends a string without quoting. Only use this for compile-time
// constants, for values already quoted for the target shell, or for
// pre-assembled flag lists that must keep their own word splitting.
func (d *DockerCmd) Raw(raw string) *DockerCmd {
	if raw == "" {
		return d
	}
	d.parts = append(d.parts, raw)

	return d
}

// String returns the assembled command line.
func (d *DockerCmd) String() string {
	return strings.Join(d.parts, " ")
}

// Quote returns the value as Arg would place it on the command line. Callers
// that redact a value out of command logging need this: the command string
// holds the quoted form, so a pattern built from the raw value does not match
// once quoting alters it.
func (d *DockerCmd) Quote(value string) string {
	return d.quote(value)
}

// newDockerCmd starts a command line with the docker executable path and the
// quoting function of the target shell.
func newDockerCmd(exe string, quote func(string) string) *DockerCmd {
	return &DockerCmd{quote: quote, parts: []string{exe}}
}

// shQuote quotes a value for a POSIX shell.
func shQuote(s string) string {
	return shellescape.Quote(s)
}

// cmdMeta are the characters cmd.exe acts on before the target process sees
// its command line.
const cmdMeta = `^&|<>()%!"`

// cmdQuote quotes a value for cmd.exe.
//
// Two parsers see the string: cmd.exe, which acts on the characters in
// cmdMeta, and the target process' CommandLineToArgvW, which acts on double
// quotes and on backslashes that precede them. The value is first quoted for
// CommandLineToArgvW, then every character cmd.exe would act on - including
// the quotes just added - is caret-escaped so cmd.exe passes it through
// literally.
func cmdQuote(s string) string {
	var argv strings.Builder

	argv.WriteByte('"')

	backslashes := 0

	for i := range len(s) {
		char := s[i]
		switch char {
		case '\\':
			backslashes++
		case '"':
			// Escape the run of backslashes preceding the quote, then the quote.
			argv.WriteString(strings.Repeat(`\`, backslashes*2+1))
			backslashes = 0
		default:
			argv.WriteString(strings.Repeat(`\`, backslashes))
			backslashes = 0
		}

		if char != '\\' {
			argv.WriteByte(char)
		}
	}

	// A trailing backslash run would escape the closing quote.
	argv.WriteString(strings.Repeat(`\`, backslashes*2))
	argv.WriteByte('"')

	quoted := argv.String()

	var escaped strings.Builder

	for i := range len(quoted) {
		char := quoted[i]
		if strings.IndexByte(cmdMeta, char) >= 0 {
			escaped.WriteByte('^')
		}
		escaped.WriteByte(char)
	}

	return escaped.String()
}
