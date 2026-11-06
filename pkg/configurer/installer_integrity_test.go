//go:build testing

package configurer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

	_, err := GetInstaller(srv.URL + "/install.ps1")

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
	// the client that trusts it, keeping the production redirect policy.
	original := installerClient
	installerClient = srv.Client()
	installerClient.CheckRedirect = original.CheckRedirect

	t.Cleanup(func() { installerClient = original })

	_, err := GetInstaller(srv.URL + "/install.ps1")

	require.ErrorIs(t, err, ErrInstallerDownloadFailed)
	require.ErrorContains(t, err, "404")
}

// The configured URL being https says nothing about where the server sends
// the request next. The default client follows an https-to-http redirect
// silently, and the script at the end of it is executed on every Windows
// host with -ExecutionPolicy Bypass.
func TestGetInstallerRejectsDowngradeRedirect(t *testing.T) {
	clearInstallerCache(t)

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(fakeScript))
	}))
	defer plain.Close()

	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	}))
	defer secure.Close()

	original := installerClient
	installerClient = secure.Client()
	installerClient.CheckRedirect = original.CheckRedirect

	t.Cleanup(func() { installerClient = original })

	_, err := GetInstaller(secure.URL + "/install.ps1")

	require.ErrorIs(t, err, ErrInstallerDownloadFailed)
	require.ErrorContains(t, err, "https")
}

// Both callers of GetInstaller are Windows, so the local-file escape hatch
// has to be spellable there. This pins which spelling works: the native
// path, which parseURL takes as a file because it contains no "://". The
// file:// forms are measured rather than assumed, because url.Parse reads
// the drive letter's colon as a port separator.
func TestGetInstallerLocalPathSpellings(t *testing.T) {
	clearInstallerCache(t)

	path := writeTempScript(t, fakeScript)

	_, err := GetInstaller(path)
	require.NoError(t, err, "the native path is the documented spelling and must work on every platform")

	if runtime.GOOS != "windows" {
		return
	}

	_, err = GetInstaller("file://" + path)
	require.Error(t, err, "file:// with a native Windows path does not parse; the docs must not offer it")

	t.Logf("file:/// with forward slashes: %v", func() error {
		_, err := GetInstaller("file:///" + filepath.ToSlash(path))

		return err
	}())
}

// writeTempScript returns a bare local path, not a file:// URL. parseURL
// treats a string without "://" as a local file, which is the spelling that
// works on both platforms: on Windows "file://C:\..." fails to parse at all
// ("invalid port" after the drive letter), so a test using it would exercise
// the URL parser rather than the verification being tested.
func writeTempScript(t *testing.T, content string) string {
	t.Helper()

	path := t.TempDir() + "/install.ps1"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}
