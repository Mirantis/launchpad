package log_test

import (
	"os"
	"path/filepath"
	"testing"

	mcclog "github.com/Mirantis/launchpad/pkg/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// logThrough writes one record at each level through a logger carrying only
// the file hook, and returns what reached the file.
func logThrough(t *testing.T, message string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "launchpad.log")

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	logger.SetOutput(os.NewFile(0, os.DevNull))
	logger.AddHook(mcclog.NewFileHook(file))

	logger.Trace(message)
	logger.Debug(message)
	logger.Info("install started")

	require.NoError(t, file.Close())

	contents, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(contents)
}

// The log file is written for every run and kept on disk, so a credential in
// a Trace record outlives the terminal it would have scrolled past. Trace and
// Debug records reach it only when the operator asked for them.
func TestFileHookPersistsTraceOnlyWhenRequested(t *testing.T) {
	const credential = "fake-admin-password"

	t.Run("default", func(t *testing.T) {
		mcclog.Debug, mcclog.Trace = false, false
		t.Cleanup(func() { mcclog.Debug, mcclog.Trace = false, false })

		got := logThrough(t, "docker login -p "+credential)

		require.NotContains(t, got, credential, "a trace record reached the log file without --debug or --trace")
		require.Contains(t, got, "install started", "the file must still carry the records it is read for")
	})

	t.Run("debug", func(t *testing.T) {
		mcclog.Debug, mcclog.Trace = true, false
		t.Cleanup(func() { mcclog.Debug, mcclog.Trace = false, false })

		got := logThrough(t, "debug record")

		require.Contains(t, got, "debug record")
	})

	t.Run("trace", func(t *testing.T) {
		mcclog.Debug, mcclog.Trace = true, true
		t.Cleanup(func() { mcclog.Debug, mcclog.Trace = false, false })

		got := logThrough(t, "trace record")

		require.Contains(t, got, "trace record")
	})
}
