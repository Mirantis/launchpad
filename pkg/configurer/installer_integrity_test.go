//go:build testing

package configurer

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const fakeScript = "# fake container runtime installer\nwrite-host hello\n"

func clearInstallerCache(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { downloadedInstallers = map[string]string{} })

	downloadedInstallers = map[string]string{}
}

// The bytes fetched here are uploaded to every Windows host and executed
// with -ExecutionPolicy Bypass, so a transport that cannot authenticate the
// server must not be used. parseURL previously accepted any scheme and
// handed it to http.Get.
func TestGetInstallerRejectsNonHTTPSURLs(t *testing.T) {
	clearInstallerCache(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fakeScript))
	}))
	defer srv.Close()

	_, err := GetInstaller(srv.URL+"/install.ps1", "")

	require.ErrorIs(t, err, ErrInstallerDownloadFailed)
	require.ErrorContains(t, err, "scheme")
}

// A 404 body is still a body. Without a status check it was written to a
// temporary file, uploaded, and executed; only the 10-byte length check
// stood in the way, and an error page is longer than ten bytes.
func TestGetInstallerRejectsNon200Status(t *testing.T) {
	clearInstallerCache(t)

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("<html><body>404: Not Found</body></html>"))
	}))
	defer srv.Close()

	// httptest's TLS server uses its own CA, so the fetch has to go through
	// the client that trusts it. http.Get reads http.DefaultClient.
	original := http.DefaultClient
	http.DefaultClient = srv.Client()
	t.Cleanup(func() { http.DefaultClient = original })
	_, err := GetInstaller(srv.URL+"/install.ps1", "")

	require.ErrorIs(t, err, ErrInstallerDownloadFailed)
	require.ErrorContains(t, err, "404")
}

// A configured pin that does not match aborts the run. There is no
// override: a pin that can be ignored is worse than no pin, because the
// configuration claims a guarantee the run does not provide.
func TestGetInstallerRejectsChecksumMismatch(t *testing.T) {
	clearInstallerCache(t)

	path := writeTempScript(t, fakeScript)

	_, err := GetInstaller("file://"+path, "0000000000000000000000000000000000000000000000000000000000000000")

	require.ErrorIs(t, err, ErrInstallerDownloadFailed)
	require.ErrorContains(t, err, "checksum")
}

// The matching pin is accepted, and the staged file holds the script.
func TestGetInstallerAcceptsMatchingChecksum(t *testing.T) {
	clearInstallerCache(t)

	path := writeTempScript(t, fakeScript)
	sum := sha256.Sum256([]byte(fakeScript))

	staged, err := GetInstaller("file://"+path, hex.EncodeToString(sum[:]))
	require.NoError(t, err)

	got, err := os.ReadFile(staged) //nolint:gosec // path produced by the code under test
	require.NoError(t, err)
	require.Equal(t, fakeScript, string(got))
}

// Two hosts configured with the same URL and different pins must not share
// a verification result: the cache key includes the checksum, so the second
// call verifies rather than returning the first call's staged file.
func TestGetInstallerCacheIsKeyedByChecksum(t *testing.T) {
	clearInstallerCache(t)

	path := writeTempScript(t, fakeScript)
	source := "file://" + path

	_, err := GetInstaller(source, "")
	require.NoError(t, err)

	_, err = GetInstaller(source, "0000000000000000000000000000000000000000000000000000000000000000")
	require.ErrorIs(t, err, ErrInstallerDownloadFailed)
}

func writeTempScript(t *testing.T, content string) string {
	t.Helper()

	path := t.TempDir() + "/install.ps1"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}
