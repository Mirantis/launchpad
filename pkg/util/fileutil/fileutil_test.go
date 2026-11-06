package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHomeDir(t *testing.T) {
	dir, err := os.UserHomeDir()
	require.NoError(t, err)

	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Expand home dir",
			path:     "~/foo",
			expected: filepath.Join(dir, "/foo"),
		},
		{
			name:     "Do not expand home dir",
			path:     "/foo",
			expected: "/foo",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ExpandHomeDir(tc.path)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, actual)
		})
	}

	t.Run("Error: cannot expand user-specific home dir", func(t *testing.T) {
		_, err := ExpandHomeDir("~foo")
		assert.ErrorIs(t, err, errCannotExpandHomeDir)
	})
}

func TestMkdirPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	t.Run("creates a new directory at 0700", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "state", "cluster")
		require.NoError(t, MkdirPrivate(dir))

		fi, err := os.Stat(dir)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o700), fi.Mode().Perm())
	})

	t.Run("narrows a directory left permissive by an earlier release", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "state")
		require.NoError(t, os.MkdirAll(dir, 0o777))
		require.NoError(t, os.Chmod(dir, 0o777))

		require.NoError(t, MkdirPrivate(dir))

		fi, err := os.Stat(dir)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o700), fi.Mode().Perm(),
			"an existing directory must be narrowed, not left as os.MkdirAll found it")
	})

	t.Run("returns the error when the directory cannot be created", func(t *testing.T) {
		// A regular file in the path makes MkdirAll fail.
		base := t.TempDir()
		blocker := filepath.Join(base, "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))

		require.Error(t, MkdirPrivate(filepath.Join(blocker, "child")))
	})

	t.Run("returns the error when the mode cannot be changed", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses the permission check this case relies on")
		}
		// A directory owned by this user inside a parent that denies search
		// cannot be chmodded, so the Chmod error must surface rather than be
		// swallowed.
		base := t.TempDir()
		parent := filepath.Join(base, "parent")
		target := filepath.Join(parent, "child")
		require.NoError(t, os.MkdirAll(target, 0o700))
		require.NoError(t, os.Chmod(parent, 0o000))
		t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

		require.Error(t, MkdirPrivate(target))
	})
}

func TestWritePrivateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	t.Run("creates a new file at 0600", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "user.yaml")
		require.NoError(t, WritePrivateFile(path, []byte("fresh")))

		fi, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), fi.Mode().Perm())

		content, err := os.ReadFile(path) // #nosec G304 -- test-controlled path
		require.NoError(t, err)
		require.Equal(t, "fresh", string(content))
	})

	t.Run("narrows a file left permissive by an earlier release", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "user.yaml")
		require.NoError(t, os.WriteFile(path, []byte("stale contents, longer"), 0o644))
		require.NoError(t, os.Chmod(path, 0o644))

		require.NoError(t, WritePrivateFile(path, []byte("new")))

		fi, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), fi.Mode().Perm(),
			"an existing file must be narrowed, not left as os.WriteFile found it")

		// Truncation must have taken effect: no tail of the longer old value.
		content, err := os.ReadFile(path) // #nosec G304 -- test-controlled path
		require.NoError(t, err)
		require.Equal(t, "new", string(content))
	})

	t.Run("returns the error when the file cannot be opened", func(t *testing.T) {
		require.Error(t, WritePrivateFile(filepath.Join(t.TempDir(), "missing", "f"), []byte("x")))
	})
}
