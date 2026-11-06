package configurer

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ps "github.com/k0sproject/rig/v2/powershell"
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

// chainPayload is the shape PRODENG-3919's acceptance criteria name: a value
// carrying a double quote and "&&", which on cmd.exe closes the current token
// and runs a second command in the administrator WinRM session.
const chainPayload = `stable" && whoami && "`

// TestCmdQuoteNeutralisesCommandChaining pins the property the Windows sites
// depend on. ps.DoubleQuote cannot do this job: it escapes a double quote as
// backtick-quote, which is a PowerShell escape that cmd.exe does not honour,
// so the quote stays live and the "&&" chains.
func TestCmdQuoteNeutralisesCommandChaining(t *testing.T) {
	got := cmdQuote(chainPayload)

	// Every cmd.exe metacharacter must be caret-escaped. An "&" that is not
	// preceded by a caret would chain.
	for i, r := range got {
		if r != '&' {
			continue
		}
		if i == 0 || got[i-1] != '^' {
			t.Fatalf("cmdQuote(%q) = %q: '&' at index %d is not caret-escaped", chainPayload, got, i)
		}
	}

	// The payload's own quote must not survive as a token-closing quote.
	if strings.Contains(got, "`\"") {
		t.Errorf("cmdQuote(%q) = %q: contains a PowerShell backtick escape, which cmd.exe does not honour", chainPayload, got)
	}

	// The whole value must still be recoverable: quoting that dropped the
	// payload would pass this test while breaking the install.
	if !strings.Contains(got, "whoami") {
		t.Errorf("cmdQuote(%q) = %q: value content was lost", chainPayload, got)
	}
}

// TestPsDoubleQuoteIsUnsafeForCmdExe records why the Windows sites moved off
// ps.DoubleQuote. It asserts the defect, so if rig ever changes the escape
// this test fails and the reasoning in the call sites can be revisited.
func TestPsDoubleQuoteIsUnsafeForCmdExe(t *testing.T) {
	got := ps.DoubleQuote(chainPayload)

	if !strings.Contains(got, "`\"") {
		t.Skipf("ps.DoubleQuote no longer emits a backtick escape (%q); revisit the cmd.exe call sites", got)
	}
	if strings.Contains(got, "^&") {
		t.Errorf("ps.DoubleQuote(%q) = %q: unexpectedly caret-escapes, which would make it cmd.exe-safe", chainPayload, got)
	}
}

// TestSetxCommandNeutralisesCommandChaining asserts the complete command
// line, not only the quoter. WinRS runs every command through cmd.exe
// (WINRS_SKIP_CMD_SHELL is sent as FALSE by masterzen/winrm, request.go:77),
// so the value is parsed by cmd.exe exactly once and must survive that parse
// as a literal.
func TestSetxCommandNeutralisesCommandChaining(t *testing.T) {
	got := setxCommand("LP3919PROBE", chainPayload)

	require.True(t, strings.HasPrefix(got, "setx "), "command must still invoke setx: %q", got)

	for i, r := range got {
		if r != '&' {
			continue
		}
		require.NotEqual(t, 0, i, "command must not begin with '&': %q", got)
		require.Equal(t, uint8('^'), got[i-1], "'&' at %d is not caret-escaped: %q", i, got)
	}

	require.NotContains(t, got, "`\"", "a PowerShell backtick escape would not be honoured by cmd.exe: %q", got)
	require.Contains(t, got, "whoami", "the value must survive quoting, not be dropped: %q", got)
}

// TestWindowsMCRInstallCommand pins the MCR installer command line (PRODENG-4062).
func TestWindowsMCRInstallCommand(t *testing.T) {
	got := windowsMCRInstallCommand(
		`C:\Users\Administrator\installer.ps1`,
		"https://repos-internal.mirantis.com",
		"test-29.9.2/fips",
		false,
	)

	require.Equal(t,
		`powershell.exe -ExecutionPolicy AllSigned -NoProfile -NonInteractive`+
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
			got := windowsMCRInstallCommand(`C:\installer.ps1`, tc.repoURL, tc.channel, false)
			require.Empty(t, unescapedCmdOperators(got), "cmd.exe would act on an operator in %q", got)
		})
	}
}

func TestWindowsMCRInstallCommandEscapesChannelQuote(t *testing.T) {
	got := windowsMCRInstallCommand(`C:\installer.ps1`, "https://repos.mirantis.com", `x" && whoami`, false)
	require.Contains(t, got, ` -Channel ^"x\^" ^&^& whoami^" -Verbose`)
}

// TestWindowsMCRInstallCommandRequiresASignedScript covers PRODENG-3924 AC-1:
// the installer runs under AllSigned, so PowerShell verifies the script's
// Authenticode signature, unless the operator opts out for a script of their
// own that Mirantis did not sign.
func TestWindowsMCRInstallCommandRequiresASignedScript(t *testing.T) {
	signed := windowsMCRInstallCommand(`C:\installer.ps1`, "https://repos.mirantis.com", "stable", false)
	require.Contains(t, signed, "-ExecutionPolicy AllSigned")
	require.NotContains(t, signed, "Bypass",
		"Bypass disables the signature check this ticket exists to restore")

	optedOut := windowsMCRInstallCommand(`C:\installer.ps1`, "https://repos.mirantis.com", "stable", true)
	require.Contains(t, optedOut, "-ExecutionPolicy Bypass",
		"installURLWindowsAllowUnsigned must still permit an operator's own script")

	// The opt-out changes the policy and nothing else.
	require.Equal(t,
		strings.Replace(signed, "-ExecutionPolicy AllSigned", "-ExecutionPolicy Bypass", 1),
		optedOut)
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
