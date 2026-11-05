package ubuntu

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// writeShim installs an executable stub on a temporary PATH so the import
// command can be run for real without fetching anything or touching the
// system keyring.
func writeShim(t *testing.T, dir, name, body string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700))
}

// injectionCases are repo URLs carrying a payload that creates the sentinel
// file INJECTED in the working directory if a shell parses the URL as code.
//
// The payload is ":>INJECTED", a shell builtin plus a redirection, not
// "touch": it must not depend on any binary being resolvable, or a PATH that
// cannot resolve the payload would make the test pass without proving
// anything. The sentinel is a bare relative name for the same reason a
// Windows temporary directory would otherwise be interpolated into the shell
// with backslashes and never match.
var injectionCases = []struct {
	name string
	url  string
}{
	{name: "command substitution", url: "https://example.invalid/$(:>INJECTED)/ubuntu/gpg"},
	{name: "backticks", url: "https://example.invalid/`:>INJECTED`/ubuntu/gpg"},
	{name: "command separator", url: "https://example.invalid/x; :>INJECTED"},
	{name: "pipeline", url: "https://example.invalid/x | :>INJECTED"},
	// Raised in review of #676: url.ParseRequestURI accepts this, so the
	// validation is not what stops it -- the positional argument is.
	{name: "IFS separated substitution", url: "https://example.invalid/$(:${IFS}>INJECTED)/ubuntu/gpg"},
}

const sentinel = "INJECTED"

// runInShell runs script through a real bash with the working directory and
// PATH pointing at dir, and reports whether the sentinel file was created.
//
// The exit status is deliberately ignored. Whether curl and gpg resolve, and
// whether they succeed, is a property of the machine running the test, not of
// the command under test; asserting on it is what made this test fail on
// Windows while passing elsewhere. Injection happens during shell expansion,
// before any of those binaries run, so the sentinel answers the question on
// its own.
func runInShell(t *testing.T, bashPath, dir, script string) bool {
	t.Helper()

	run := exec.Command(bashPath, "-c", script) //nolint:gosec // the command under test is the point
	run.Dir = dir
	// The stub directory alone, never prepended to the real PATH: a stub that
	// fails to resolve then fails loudly instead of silently running the
	// machine's real curl against a URL and its real gpg against
	// /usr/share/keyrings.
	run.Env = append(os.Environ(), "PATH="+dir)
	out, _ := run.CombinedOutput()
	t.Logf("%s\n%s", script, out)

	_, err := os.Stat(filepath.Join(dir, sentinel))

	return err == nil
}

func shimmedDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	writeShim(t, dir, "curl", "#!/bin/sh\necho ARMOURED\n")
	// gpg must drain stdin so the pipeline exits cleanly.
	writeShim(t, dir, "gpg", "#!/bin/sh\ncat > /dev/null\n")

	return dir
}

// TestGPGImportCommandDoesNotExecuteURL runs the generated command through a
// real bash and proves that shell metacharacters in spec.mcr.repoURL are
// fetched as data instead of starting a second command. This is the injection
// that reached a root shell: the URL is passed to bash as a positional
// argument, so bash never parses it as code.
func TestGPGImportCommandDoesNotExecuteURL(t *testing.T) {
	t.Parallel()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}

	for _, tc := range injectionCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			injected := runInShell(t, bashPath, shimmedDir(t), gpgImportCommand(tc.url))
			require.False(t, injected, "the repo URL was executed as a command")
		})
	}
}

// TestInjectionPayloadsAreDetectable is the positive control for the test
// above: it runs the vulnerable shape, the URL interpolated into the script
// text as the code this command used to build, and requires every payload to
// create the sentinel. Without it, a payload that silently stopped working on
// some platform would leave TestGPGImportCommandDoesNotExecuteURL passing
// while proving nothing.
func TestInjectionPayloadsAreDetectable(t *testing.T) {
	t.Parallel()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}

	for _, tc := range injectionCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vulnerable := "set -o pipefail; curl -fsSL -- " + tc.url + " | gpg --batch --yes --output keyring.gpg --dearmor"
			injected := runInShell(t, bashPath, shimmedDir(t), vulnerable)
			require.True(t, injected, "payload did not execute in the vulnerable shape, so the test above proves nothing")
		})
	}
}

// TestGPGImportCommandKeepsURLOutOfScript pins the mechanism the test above
// verifies behaviourally: the script text is a constant that refers to the URL
// as the quoted positional "$1", so no URL value can ever be interpolated into
// code.
func TestGPGImportCommandKeepsURLOutOfScript(t *testing.T) {
	t.Parallel()

	const gpgURL = "https://example.invalid/$(id)/ubuntu/gpg"

	require.Contains(t, gpgKeyImportScript, `"$1"`)
	require.NotContains(t, gpgKeyImportScript, gpgURL)

	got := gpgImportCommand(gpgURL)
	require.Contains(t, got, "bash -c ")
	// The URL is a separate, quoted argument after the script and its $0.
	require.Contains(t, got, "launchpad-mcr-gpg")
	require.True(t, strings.HasSuffix(got, `'https://example.invalid/$(id)/ubuntu/gpg'`),
		"URL should be the trailing single-quoted argument, got %q", got)
}
