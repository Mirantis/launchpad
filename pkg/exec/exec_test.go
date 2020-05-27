package exec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExec_Stdin(t *testing.T) {
	opts := options{}
	Stdin("hello")(&opts)
	require.Equal(t, "hello", opts.Stdin)
}

func TestExec_StreamOutput(t *testing.T) {
	opts := options{}
	StreamOutput()(&opts)
	require.True(t, opts.LogInfo)
}

func TestExec_HideCommand(t *testing.T) {
	opts := options{}
	StreamOutput()(&opts)
	require.False(t, opts.LogCommand)
}

func TestExec_HideOutput(t *testing.T) {
	opts := options{}
	HideOutput()(&opts)
	require.False(t, opts.LogDebug)
}

func TestExec_Redact(t *testing.T) {
	opts := options{}
	Redact("hello")(&opts)
	require.Equal(t, "hello", opts.Redact)
}

func TestExec_redactFunc(t *testing.T) {
	f := redactFunc("hello")
	require.Equal(t, "[REDACTED] world", f("hello world"))
	f = redactFunc("(?i)hello")
	require.Equal(t, "[REDACTED] world", f("Hello world"))
}
