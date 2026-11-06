// Package secretenv keeps secret values out of remote command lines.
//
// A value placed in a command's arguments is readable on the managed host for
// as long as the process lives: /proc/<pid>/cmdline is world readable, auditd
// records the argv of every execve, and shell history, journald and Windows
// 4688 events keep their own copies. rig's cmd.Redact only masks launchpad's
// own logging and has no effect on any of those.
//
// The MKE and MSR bootstrappers accept every secret they take as a flag
// through an environment variable instead (MKE 3.3-3.9 and MSR 2.8-2.9 CLI
// references). This package stages those values in mode-0600 files on the
// host and builds the shell assignments that read them back, so the command
// line carries only variable names and paths:
//
//	UCP_ADMIN_PASSWORD="$(cat -- '/tmp/launchpad-env-abc')" /usr/bin/docker run -e UCP_ADMIN_PASSWORD ...
//
// The environment of a running process is readable only by its owner and
// root, unlike its command line, and execve auditing does not record it.
//
// docker's own --env-file is deliberately not used: that format is one
// KEY=VALUE per line and cannot carry a multi-line value, which is exactly
// what the PEM-encoded MSR_KEY, MSR_CERT and UCP_CA values are.
//
// Staging is POSIX only. Both bootstrappers run as Linux containers on a
// manager node, so no Windows path reaches this package.
package secretenv

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/k0sproject/rig/v2/remotefs"
	"github.com/k0sproject/rig/v2/sh/shellescape"
)

// fileMode is the mode of a staged secret file. rig's PosixFS.WriteFile
// creates the file with this mode (install -D -m), so there is no window in
// which the secret exists with wider permissions.
const fileMode fs.FileMode = 0o600

// filePrefix names staged files recognisably, so an operator finding one
// after a crash knows what it is and which tool left it.
const filePrefix = "launchpad-env-"

// Host is the part of a managed host this package needs.
type Host interface {
	FS() remotefs.FS
}

// Vars is a set of secret values staged on a host.
type Vars struct {
	host  Host
	names []string
	paths map[string]string
}

// Stage writes each value to its own mode-0600 file on the host. The caller
// must call Remove when the command that consumes them has finished, on every
// path including failures: a staged secret left behind outlives the run.
//
// Staging stops at the first failure and removes whatever was already
// written. The caller gets no handle back in that case, so a failure to clean
// up is reported alongside the staging failure rather than dropped: it means
// a credential is sitting on the host that nothing will remove.
func Stage(host Host, values map[string]string) (*Vars, error) {
	vars := &Vars{host: host, paths: make(map[string]string, len(values))}

	for _, name := range sortedNames(values) {
		// mktemp creates the file itself, so the path cannot be predicted
		// and pointed at something else between naming and writing.
		path, err := host.FS().CreateTemp("", filePrefix)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("failed to create a file for %s: %w", name, err), vars.Remove())
		}

		vars.names = append(vars.names, name)
		vars.paths[name] = path

		if err := host.FS().WriteFile(path, []byte(values[name]), fileMode); err != nil {
			return nil, errors.Join(fmt.Errorf("failed to write the value of %s: %w", name, err), vars.Remove())
		}
	}

	return vars, nil
}

// Assignments returns the shell variable assignments that prefix the command,
// or "" when nothing is staged. The values are read on the host at execution
// time, so no secret appears in the command line.
func (v *Vars) Assignments() string {
	if v == nil || len(v.names) == 0 {
		return ""
	}

	parts := make([]string, 0, len(v.names))
	for _, name := range v.names {
		parts = append(parts, fmt.Sprintf(`%s="$(cat -- %s)"`, name, shellescape.Quote(v.paths[name])))
	}

	return strings.Join(parts, " ")
}

// PassFlags returns the docker run flags that pass the staged variables into
// the container by name, taking their values from the environment docker
// itself runs with.
func (v *Vars) PassFlags() []string {
	if v == nil {
		return nil
	}

	flags := make([]string, 0, len(v.names))
	for _, name := range v.names {
		flags = append(flags, "-e "+name)
	}

	return flags
}

// Names returns the staged variable names in command order.
func (v *Vars) Names() []string {
	if v == nil {
		return nil
	}

	return append([]string(nil), v.names...)
}

// Remove deletes every staged file. It attempts all of them and reports the
// failures together, so one undeletable file does not strand the rest.
func (v *Vars) Remove() error {
	if v == nil {
		return nil
	}

	var errs []error

	for _, name := range v.names {
		if err := v.host.FS().Remove(v.paths[name]); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove the staged value of %s: %w", name, err))
		}
	}

	v.names = nil
	v.paths = map[string]string{}

	return errors.Join(errs...)
}

// Extract moves the values of credential-bearing flags out of a flag list.
// It returns the environment values keyed by variable name and the flags that
// remain. envByFlag maps a flag name to the environment variable the
// bootstrapper reads it from.
//
// Flags.Join concatenates the list with spaces before it reaches a shell, so
// a credential can be written either inline ("--admin-password secret",
// "--admin-password=secret") or as a bare flag whose value is the next
// element. Both forms are consumed; missing the second would leave the
// credential in argv.
//
// A flag with an empty value is left alone: it carries no secret, and
// removing it would change what the bootstrapper is asked to do.
//
// The one case this cannot reach: a bare credential flag whose value is the
// next element and itself begins with "-", which is indistinguishable from
// the next flag. Such a value stays in the command line, silently. The rest
// of the codebase shares the assumption -- Flags.Index and Flags.GetValue
// read one flag and its value per element -- so a value of that shape has
// never worked through this list anyway.
func Extract(flags []string, envByFlag map[string]string) (map[string]string, []string) {
	values := map[string]string{}
	kept := make([]string, 0, len(flags))

	for i := 0; i < len(flags); i++ {
		name, value, hasValue := splitFlag(flags[i])

		env, isSecret := envByFlag[name]
		if !isSecret {
			kept = append(kept, flags[i])

			continue
		}

		// A bare credential flag takes the next element as its value.
		if !hasValue && i+1 < len(flags) {
			if _, _, nextHasValue := splitFlag(flags[i+1]); !nextHasValue && !strings.HasPrefix(flags[i+1], "-") {
				values[env] = unquote(flags[i+1])
				i++

				continue
			}
		}

		if value == "" {
			kept = append(kept, flags[i])

			continue
		}

		values[env] = value
	}

	return values, kept
}

// splitFlag separates "--flag=value" or "--flag value" into its name and its
// shell-decoded value.
func splitFlag(flag string) (name, value string, hasValue bool) {
	idx := strings.IndexAny(flag, "= ")
	if idx < 0 {
		return flag, "", false
	}

	return flag[:idx], unquote(strings.TrimSpace(flag[idx+1:])), true
}

// unquote decodes one shell word as a POSIX shell would, without performing
// any expansion. A flag list is written by hand and then joined into a shell
// command, so a value may be single quoted, double quoted, backslash escaped
// or a concatenation of those: --registry-password="pa\$word" reached the
// registry as pa$word before these values were taken out of the command
// line, and must still do so now.
//
// Expansions are deliberately not performed: $(...) and $VAR were expanded by
// the shell before, and reproducing that here would execute host-supplied
// text in launchpad. They are left as written, which is the safe difference.
//
// One deviation from a real shell: a word ending in an unescaped backslash
// escapes the following space when the flag list is joined, so the shell
// would merge the next flag into this value. That is not reachable from a
// shellescape.Quote'd value, and the pre-existing behaviour was worse -- the
// merged word went to the bootstrapper as one argument.
func unquote(value string) string {
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

func sortedNames(values map[string]string) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
