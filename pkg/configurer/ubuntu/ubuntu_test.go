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

// TestGPGImportCommandDoesNotExecuteURL runs the generated command through a
// real bash with curl and gpg stubbed out, and proves that shell
// metacharacters in spec.mcr.repoURL are fetched as data instead of starting a
// second command. This is the injection that reached a root shell: the URL is
// passed to bash as a positional argument, so bash never parses it as code.
func TestGPGImportCommandDoesNotExecuteURL(t *testing.T) {
	t.Parallel()

	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}

	for _, tc := range []struct {
		name string
		urlf string
	}{
		{name: "command substitution", urlf: "https://example.invalid/$(touch %s)/ubuntu/gpg"},
		{name: "backticks", urlf: "https://example.invalid/`touch %s`/ubuntu/gpg"},
		{name: "command separator", urlf: "https://example.invalid/x; touch %s"},
		{name: "pipeline", urlf: "https://example.invalid/x | touch %s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			marker := filepath.Join(dir, "INJECTED")
			argsFile := filepath.Join(dir, "curl-args")
			writeShim(t, dir, "curl", "#!/bin/sh\nprintf '%s\\n' \"$@\" > "+argsFile+"\necho ARMOURED\n")
			// gpg must drain stdin so the pipeline exits cleanly.
			writeShim(t, dir, "gpg", "#!/bin/sh\ncat > /dev/null\n")

			gpgURL := strings.Replace(tc.urlf, "%s", marker, 1)

			run := exec.Command(bashPath, "-c", gpgImportCommand(gpgURL)) //nolint:gosec // the command under test is the point
			run.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			out, runErr := run.CombinedOutput()
			require.NoError(t, runErr, "command failed: %s", out)

			require.NoFileExists(t, marker, "the repo URL was executed as a command")

			recorded, readErr := os.ReadFile(argsFile)
			require.NoError(t, readErr)
			require.Equal(t,
				[]string{"-fsSL", "--", gpgURL},
				strings.Split(strings.TrimRight(string(recorded), "\n"), "\n"),
				"the URL must arrive as a single literal curl argument")
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
