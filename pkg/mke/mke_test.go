package mke

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
)

func TestVersionGreaterThan(t *testing.T) {
	pairs := [][]string{
		{"1.0.0-beta2", "1.0.0-beta1"},
		{"1.0.0", "1.0.0-rc1"},
		{"1.0.0-rc1", "1.0.0-tp3"},
	}

	for _, pair := range pairs {
		va, _ := version.NewVersion(pair[0])
		vb, _ := version.NewVersion(pair[1])
		require.Truef(t, VersionGreaterThan(va, vb), "%s should be greater than %s", va, vb)
	}
}

func TestSafePathRejectsEscapes(t *testing.T) {
	base := t.TempDir()
	// The real bundle dir is .../bundle/<adminUsername>, so a sibling
	// directory sharing the admin name as a prefix is the realistic escape.
	admin := filepath.Join(base, "bundle", "admin")

	tests := []struct {
		name string
		rel  string
		want bool // true when the path must be rejected
	}{
		{name: "plain member", rel: "kube.yml", want: false},
		{name: "member in a subdirectory", rel: "certs/cert.pem", want: false},
		{name: "parent escape", rel: "../evil", want: true},
		{name: "deep parent escape", rel: "../../../etc/evil", want: true},
		// The bypass: filepath.Join resolves this to .../bundle/adminX/evil,
		// which a bare strings.HasPrefix against .../bundle/admin accepts
		// because the sibling's name starts with the base's name.
		{name: "sibling directory sharing the base name prefix", rel: "../adminX/evil", want: true},
		// filepath.Join strips the leading separator, so an absolute member
		// name lands inside the bundle dir rather than escaping it.
		{name: "absolute path member is contained, not an escape", rel: "/etc/evil", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safePath(admin, tt.rel)
			if tt.want {
				require.ErrorIs(t, err, errInvalidBundle, "path %q must be rejected, got %q", tt.rel, got)
				return
			}
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(got, admin+string(os.PathSeparator)),
				"accepted path %q must stay under %q", got, admin)
		})
	}
}

// buildBundle produces a zip with the member names a real MKE client bundle
// carries, including one inside a subdirectory.
func buildBundle(t *testing.T, names []string) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte("contents of " + name))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	return zr
}

func TestWriteBundlePermissions(t *testing.T) {
	// Windows does not carry POSIX mode bits, so the assertions below are
	// meaningless there; the unit suite runs on windows-latest in CI.
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}

	// Modes are asserted as they land on disk, not as the constant passed to
	// MkdirAll. umask can only clear bits, so it cannot turn 0700 or 0600
	// into something more permissive, but a regression that passed a wider
	// mode would show up here and not in a constant check.

	names := []string{"ca.pem", "cert.pem", "key.pem", "kube.yml", "env.sh", "certs/extra.pem"}
	bundleDir := filepath.Join(t.TempDir(), "bundle", "admin")

	require.NoError(t, writeBundle(bundleDir, buildBundle(t, names)))

	di, err := os.Stat(bundleDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), di.Mode().Perm(), "bundle directory must not be group or world accessible")

	sub, err := os.Stat(filepath.Join(bundleDir, "certs"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o700), sub.Mode().Perm(), "bundle subdirectory must not be group or world accessible")

	for _, name := range names {
		fi, err := os.Stat(filepath.Join(bundleDir, name))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), fi.Mode().Perm(),
			"%s must not be readable by group or other", name)
	}
}

func TestWriteBundleRejectsEscapingMember(t *testing.T) {
	base := t.TempDir()
	bundleDir := filepath.Join(base, "bundle", "admin")

	err := writeBundle(bundleDir, buildBundle(t, []string{"../adminX/evil"}))
	require.ErrorIs(t, err, errInvalidBundle)

	// Nothing may have been written outside the bundle directory.
	_, statErr := os.Stat(filepath.Join(base, "bundle", "adminX"))
	require.ErrorIs(t, statErr, os.ErrNotExist, "escaping member must not create a sibling directory")
}
