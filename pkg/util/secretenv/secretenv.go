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
// written, so a partial failure leaves nothing behind.
func Stage(host Host, values map[string]string) (*Vars, error) {
	vars := &Vars{host: host, paths: make(map[string]string, len(values))}

	for _, name := range sortedNames(values) {
		// mktemp creates the file itself, so the path cannot be predicted
		// and pointed at something else between naming and writing.
		path, err := host.FS().CreateTemp("", filePrefix)
		if err != nil {
			_ = vars.Remove()

			return nil, fmt.Errorf("failed to create a file for %s: %w", name, err)
		}

		vars.names = append(vars.names, name)
		vars.paths[name] = path

		if err := host.FS().WriteFile(path, []byte(values[name]), fileMode); err != nil {
			_ = vars.Remove()

			return nil, fmt.Errorf("failed to write the value of %s: %w", name, err)
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
// A flag with an empty value is left alone: it carries no secret, and
// removing it would change what the bootstrapper is asked to do.
func Extract(flags []string, envByFlag map[string]string) (map[string]string, []string) {
	values := map[string]string{}
	kept := make([]string, 0, len(flags))

	for _, flag := range flags {
		name, value, found := splitFlag(flag)

		env, isSecret := envByFlag[name]
		if !found || !isSecret || value == "" {
			kept = append(kept, flag)

			continue
		}

		values[env] = value
	}

	return values, kept
}

// splitFlag separates "--flag=value" or "--flag value" into its name and
// value, unquoting the value. Single quotes are handled here because
// strconv.Unquote, which config.FlagValue uses, does not treat a
// single-quoted string as a string.
func splitFlag(flag string) (name, value string, found bool) {
	idx := strings.IndexAny(flag, "= ")
	if idx < 0 {
		return flag, "", false
	}

	name = flag[:idx]
	value = strings.TrimSpace(flag[idx+1:])

	if len(value) > 1 {
		first, last := value[0], value[len(value)-1]
		if first == last && (first == '\'' || first == '"') {
			value = value[1 : len(value)-1]
		}
	}

	return name, value, true
}

func sortedNames(values map[string]string) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
