package configurer

import (
	"context"
	"errors"
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
