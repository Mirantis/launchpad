package configurer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestIsExitCode3010 pins recognition of the Windows
// ERROR_SUCCESS_REBOOT_REQUIRED exit code across the differing error
// phrasings used by rig v1 (SSH/WinRM) and rig v2 WinRM, plus other
// unrelated exit codes and errors that must not be mistaken for it.
func TestIsExitCode3010(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "rig v1 phrasing",
			err:  errors.New("command result: process finished with error: non-zero exit code: 3010"),
			want: true,
		},
		{
			name: "rig v2 winrm phrasing",
			err:  errors.New("command result: process finished with error: command exited with a non-zero exit code: exit code 3010"),
			want: true,
		},
		{
			name: "unrelated non-zero exit code",
			err:  errors.New("command exited with a non-zero exit code: exit code 1"),
			want: false,
		},
		{
			name: "exit code containing 3010 as a substring but not the value",
			err:  errors.New("command exited with a non-zero exit code: exit code 13010"),
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection reset by peer"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isExitCode3010(tt.err))
		})
	}
}

// TestExecCtxIsBoundedAndCancelable pins the two properties the MCR
// install/uninstall/restart lifecycle relies on: execCtx() actually carries
// a deadline (unlike context.Background(), which never times out and so
// can't bound a command.Wait() call, see k0sproject/rig#472) and calling the
// returned cancel func actually cancels it rather than being a no-op.
func TestExecCtxIsBoundedAndCancelable(t *testing.T) {
	ctx, cancel := execCtx()
	defer cancel()

	deadline, ok := ctx.Deadline()
	require.True(t, ok, "execCtx() context must carry a deadline")
	require.WithinDuration(t, time.Now().Add(windowsExecTimeout), deadline, time.Second)

	require.NoError(t, ctx.Err())
	cancel()
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}

// TestWindowsMCRInstallCommand pins the MCR installer command line (PRODENG-4062).
func TestWindowsMCRInstallCommand(t *testing.T) {
	got := windowsMCRInstallCommand(
		`C:\Users\Administrator\installer.ps1`,
		"https://repos-internal.mirantis.com",
		"test-29.9.2/fips",
	)

	require.Equal(t,
		`powershell.exe -ExecutionPolicy Bypass -NoProfile -NonInteractive`+
			` -File ^"C:\Users\Administrator\installer.ps1^"`+
			` -DownloadUrl ^"https://repos-internal.mirantis.com^"`+
			` -Channel ^"test-29.9.2/fips^"`+
			` -Verbose`,
		got)
	require.Empty(t, unescapedCmdOperators(got))
}

// TestWindowsMCRInstallCommandDoesNotChainCommands covers PRODENG-3919.
func TestWindowsMCRInstallCommandDoesNotChainCommands(t *testing.T) {
	for _, tc := range []struct {
		name    string
		repoURL string
		channel string
	}{
		{name: "quote and && in channel", repoURL: "https://repos.mirantis.com", channel: `stable" && whoami && "`},
		{name: "pipe in channel", repoURL: "https://repos.mirantis.com", channel: "stable | whoami"},
		{name: "redirect in repo URL", repoURL: `https://repos.mirantis.com" > C:\pwned.txt`, channel: "stable"},
		{name: "ampersand in repo URL query", repoURL: "https://mirror.example.com/?a=1&b=2", channel: "stable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := windowsMCRInstallCommand(`C:\installer.ps1`, tc.repoURL, tc.channel)
			require.Empty(t, unescapedCmdOperators(got), "cmd.exe would act on an operator in %q", got)
		})
	}
}

func TestWindowsMCRInstallCommandEscapesChannelQuote(t *testing.T) {
	got := windowsMCRInstallCommand(`C:\installer.ps1`, "https://repos.mirantis.com", `x" && whoami`)
	require.Contains(t, got, ` -Channel ^"x\^" ^&^& whoami^" -Verbose`)
}

// unescapedCmdOperators returns the &, |, < and > that cmd.exe would act on.
func unescapedCmdOperators(line string) []string {
	var ops []string

	inQuote := false

	for i := 0; i < len(line); i++ {
		switch char := line[i]; {
		case char == '^' && !inQuote:
			i++
		case char == '"':
			inQuote = !inQuote
		case !inQuote && strings.IndexByte("&|<>", char) >= 0:
			ops = append(ops, string(char))
		}
	}

	return ops
}
