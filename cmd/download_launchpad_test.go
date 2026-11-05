//go:build testing

package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/version"
	"github.com/stretchr/testify/require"
)

// assetServer serves one asset body and points the allowlist at itself, so
// the verification logic can be exercised without reaching GitHub.
func assetServer(t *testing.T, body []byte, status int) *httptest.Server {
	t.Helper()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	host, err := url.Parse(srv.URL)
	require.NoError(t, err)

	originalHosts := assetHosts
	originalClient := assetClient
	assetHosts = []string{host.Hostname()}
	assetClient = srv.Client()

	t.Cleanup(func() {
		assetHosts = originalHosts
		assetClient = originalClient
	})

	return srv
}

func digestOf(body []byte) string {
	sum := sha256.Sum256(body)

	return hex.EncodeToString(sum[:])
}

// The URL comes from the GitHub API response, which is remote input that
// names where an executable is fetched from. A host outside GitHub's
// release hosts is refused before any request is made.
func TestFetchAssetRejectsUntrustedHost(t *testing.T) {
	_, err := fetchAsset("https://example.invalid/launchpad")

	require.ErrorIs(t, err, errUntrustedAssetHost)
}

func TestFetchAssetRequiresHTTPS(t *testing.T) {
	_, err := fetchAsset("http://github.com/Mirantis/launchpad/releases/download/1.0.0/launchpad")

	require.ErrorIs(t, err, errUntrustedAssetHost)
}

// A 404 body was previously written to launchpad and chmod 0755: the error
// page became the executable, with no error reported.
func TestDownloadVerifiedRejectsNon200(t *testing.T) {
	srv := assetServer(t, []byte("404: Not Found"), http.StatusNotFound)
	dest := filepath.Join(t.TempDir(), "launchpad")

	err := downloadVerified(&version.Asset{Name: "launchpad", URL: srv.URL}, digestOf([]byte("x")), dest)

	require.ErrorIs(t, err, errAssetHTTPStatus)
	require.NoFileExists(t, dest, "a failed download must not leave a file behind")
}

// A mismatched digest aborts and leaves nothing executable on disk.
func TestDownloadVerifiedRejectsDigestMismatch(t *testing.T) {
	srv := assetServer(t, []byte("tampered binary"), http.StatusOK)
	dest := filepath.Join(t.TempDir(), "launchpad")

	err := downloadVerified(&version.Asset{Name: "launchpad", URL: srv.URL}, digestOf([]byte("expected binary")), dest)

	require.ErrorIs(t, err, errChecksumMismatch)
	require.NoFileExists(t, dest)
}

// The matching digest is written into place, executable.
func TestDownloadVerifiedWritesVerifiedAsset(t *testing.T) {
	body := []byte("#!/bin/sh\necho launchpad\n")
	srv := assetServer(t, body, http.StatusOK)
	dest := filepath.Join(t.TempDir(), "launchpad")

	require.NoError(t, downloadVerified(&version.Asset{Name: "launchpad", URL: srv.URL}, digestOf(body), dest))

	got, err := os.ReadFile(dest) //nolint:gosec // path created by the test
	require.NoError(t, err)
	require.Equal(t, body, got)

	// Windows has no POSIX permission bits: os.Stat reports 0666 or 0444
	// there and os.Chmod cannot set the executable bit, so this assertion
	// would fail on a correct implementation.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(dest)
		require.NoError(t, err)
		require.NotZero(t, info.Mode().Perm()&0o100, "the downloaded binary must be executable")
	}
}

// A redirect is chosen by the server, not by the caller, so validating only
// the initial URL leaves the transport to whoever answers. Both the binary
// and the checksums file go through this client: a downgrade applying to
// both would let an attacker replace the binary and the digest it is
// checked against together, and the verification would still pass.
func TestDownloadVerifiedRejectsDowngradeRedirect(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("attacker binary"))
	}))
	t.Cleanup(plain.Close)

	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL, http.StatusFound)
	}))
	t.Cleanup(secure.Close)

	secureHost, err := url.Parse(secure.URL)
	require.NoError(t, err)
	plainHost, err := url.Parse(plain.URL)
	require.NoError(t, err)

	originalHosts, originalClient := assetHosts, assetClient
	// Both hosts are allowlisted, so only the scheme check can stop this.
	assetHosts = []string{secureHost.Hostname(), plainHost.Hostname()}
	assetClient = secure.Client()
	assetClient.CheckRedirect = originalClient.CheckRedirect

	t.Cleanup(func() { assetHosts, assetClient = originalHosts, originalClient })

	dest := filepath.Join(t.TempDir(), "launchpad")
	err = downloadVerified(&version.Asset{Name: "launchpad", URL: secure.URL}, digestOf([]byte("attacker binary")), dest)

	require.ErrorIs(t, err, errUntrustedAssetHost)
	require.NoFileExists(t, dest)
}

// The defect O_TRUNC hid: overwriting a longer existing binary with a
// shorter one left the tail of the old file in place, producing a corrupt
// executable from a successful download. Staging to a temporary file and
// renaming makes the result independent of what was there before.
func TestDownloadVerifiedReplacesLongerExistingBinary(t *testing.T) {
	body := []byte("short new binary\n")
	srv := assetServer(t, body, http.StatusOK)

	dest := filepath.Join(t.TempDir(), "launchpad")
	require.NoError(t, os.WriteFile(dest, []byte(strings.Repeat("old binary contents\n", 100)), 0o755)) //nolint:gosec // mirrors the real file mode

	require.NoError(t, downloadVerified(&version.Asset{Name: "launchpad", URL: srv.URL}, digestOf(body), dest))

	got, err := os.ReadFile(dest) //nolint:gosec // path created by the test
	require.NoError(t, err)
	require.Equal(t, body, got, "the old binary's tail survived the overwrite")
}

// The release publishes one combined checksums.sha256 in sha256sum format,
// not a per-asset sibling, and AssetForHost's .sha256 filter excludes it
// from the download candidates.
func TestPublishedDigestReadsCombinedChecksumsFile(t *testing.T) {
	assetName := "launchpad_linux_amd64_9.9.9"
	want := digestOf([]byte("binary"))
	checksums := fmt.Sprintf("%s  other_asset\n%s  %s\n", digestOf([]byte("other")), want, assetName)

	srv := assetServer(t, []byte(checksums), http.StatusOK)

	latest := &version.LaunchpadRelease{
		TagName: "9.9.9",
		Assets: []version.Asset{
			{Name: assetName, URL: srv.URL + "/asset"},
			{Name: version.ChecksumsAssetName, URL: srv.URL + "/checksums"},
		},
	}

	got, err := publishedDigest(latest, assetName)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// An asset with no published digest cannot be verified, so the download
// aborts rather than writing an unverified executable.
func TestPublishedDigestFailsWhenAssetIsNotListed(t *testing.T) {
	srv := assetServer(t, []byte(digestOf([]byte("other"))+"  other_asset\n"), http.StatusOK)

	latest := &version.LaunchpadRelease{
		TagName: "9.9.9",
		Assets:  []version.Asset{{Name: version.ChecksumsAssetName, URL: srv.URL}},
	}

	_, err := publishedDigest(latest, "launchpad_linux_amd64_9.9.9")

	require.ErrorIs(t, err, version.ErrNoChecksum)
}

func TestPublishedDigestFailsWhenReleaseHasNoChecksums(t *testing.T) {
	latest := &version.LaunchpadRelease{
		TagName: "9.9.9",
		Assets:  []version.Asset{{Name: "launchpad_linux_amd64_9.9.9", URL: "https://github.com/a"}},
	}

	_, err := publishedDigest(latest, "launchpad_linux_amd64_9.9.9")

	require.ErrorIs(t, err, version.ErrNoChecksum)
}
