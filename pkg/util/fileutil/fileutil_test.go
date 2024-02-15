package fileutil

import (
	"os"
	"path/filepath"
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
