package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mirantis/launchpad/version"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

var (
	errNoDownloadAvailable = fmt.Errorf("no download available")

	// errUntrustedAssetHost rejects a download URL that does not belong to
	// GitHub's release hosts. The URL arrives from the GitHub API response,
	// which is why it was previously taken on trust, but the response is
	// still remote input and it names where an executable is fetched from.
	errUntrustedAssetHost = errors.New("release asset is not hosted on a GitHub release host")

	// errChecksumMismatch aborts when the file on disk does not hash to the
	// published digest.
	errChecksumMismatch = errors.New("downloaded launchpad binary does not match its published checksum")

	// errAssetHTTPStatus aborts on a non-200 response. Without this, the
	// body of an error page was written to launchpad and made executable.
	errAssetHTTPStatus = errors.New("unexpected HTTP status downloading release asset")
)

// assetHosts are the hosts GitHub serves release assets from.
// browser_download_url points at github.com, which redirects to the object
// store, so both are needed: the allowlist is applied to the initial URL and
// to every redirect hop.
var assetHosts = []string{
	"github.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
}

func isAssetHost(host string) bool {
	for _, allowed := range assetHosts {
		if strings.EqualFold(host, allowed) {
			return true
		}
	}

	return false
}

// assetClient follows redirects only while they stay on https and on
// GitHub's release hosts, so a redirect can move the download neither to an
// arbitrary server nor onto a transport that cannot authenticate the one it
// names.
//
// The scheme check matters as much as the host check: both the binary and
// the checksums file are fetched through this client, so a downgrade that
// applied to both would let an on-path attacker substitute the binary and
// the digest it is compared against, and the verification would pass.
var assetClient = &http.Client{
	Timeout: 10 * time.Minute,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}

		if req.URL.Scheme != "https" || !isAssetHost(req.URL.Hostname()) {
			return fmt.Errorf("%w: redirected to %s", errUntrustedAssetHost, req.URL.Redacted())
		}

		return nil
	},
}

// fetchAsset retrieves a release asset after checking its host and status.
func fetchAsset(rawURL string) (*http.Response, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse release asset URL: %w", err)
	}

	if parsed.Scheme != "https" || !isAssetHost(parsed.Hostname()) {
		return nil, fmt.Errorf("%w: %s", errUntrustedAssetHost, rawURL)
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := assetClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform download request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()

		return nil, fmt.Errorf("%w: %s returned %s", errAssetHTTPStatus, rawURL, resp.Status)
	}

	return resp, nil
}

// publishedDigest returns the SHA-256 the release publishes for the asset.
//
// A release with no checksums file, or a checksums file that does not list
// the asset, aborts the download. There is no fall back to writing the
// binary unverified: the whole point of the check is that an operator who
// runs download-launchpad gets an executable whose bytes were vouched for.
func publishedDigest(latest *version.LaunchpadRelease, assetName string) (string, error) {
	checksums := latest.ChecksumsAsset()
	if checksums == nil {
		return "", fmt.Errorf("%w: release %s publishes no %s", version.ErrNoChecksum, latest.TagName, version.ChecksumsAssetName)
	}

	resp, err := fetchAsset(checksums.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read the published checksums: %w", err)
	}

	digest, err := version.DigestFor(body, assetName)
	if err != nil {
		return "", fmt.Errorf("failed to find the published checksum: %w", err)
	}
	return digest, nil
}

// downloadVerified writes the asset to a temporary file beside dest, hashes
// what landed on disk, and renames it into place only if the digest matches.
//
// The hash is computed by reading the file back rather than by teeing the
// response, because the two are not the same claim: the response bytes being
// correct says nothing about what the file contains. The previous code
// opened dest with os.O_CREATE|os.O_WRONLY and no os.O_TRUNC, so a shorter
// new binary left the tail of the old one in place and produced a corrupt
// executable from a successful download -- and a digest checked against the
// response would have passed anyway.
func downloadVerified(asset *version.Asset, digest, dest string) error {
	resp, err := fetchAsset(asset.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dir := filepath.Dir(dest)

	tmp, err := os.CreateTemp(dir, filepath.Base(dest)+".download-*")
	if err != nil {
		return fmt.Errorf("failed to create a temporary file for the download: %w", err)
	}

	tmpName := tmp.Name()

	// Any exit before the rename removes the partial file: an unverified
	// executable must not survive the command that rejected it.
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading")

	if _, err := io.Copy(io.MultiWriter(tmp, bar), resp.Body); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to finish writing the download: %w", err)
	}

	written, err := fileDigest(tmpName)
	if err != nil {
		return err
	}

	if !strings.EqualFold(written, digest) {
		return fmt.Errorf("%w: expected %s, got %s", errChecksumMismatch, digest, written)
	}

	if err := os.Chmod(tmpName, 0o755); err != nil {
		return fmt.Errorf("failed to make the download executable: %w", err)
	}

	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("failed to move the verified download into place: %w", err)
	}

	return nil
}

// fileDigest hashes the bytes that are actually on disk.
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open the download for verification: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to read the download for verification: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// NewDownloadLaunchpadCommand creates new 'download-launchpad' command to be called from cli.
func NewDownloadLaunchpadCommand() *cli.Command {
	return &cli.Command{
		Name:  "download-launchpad",
		Usage: "Download the latest launchpad version",
		Action: func(_ *cli.Context) error {
			latest, err := version.GetLatest(time.Second * 20)
			if err != nil {
				return fmt.Errorf("failed to look up the latest release: %w", err)
			}
			if !latest.IsNewer() {
				return fmt.Errorf("%w: upgrade not available", errNoDownloadAvailable)
			}
			asset := latest.AssetForHost()
			if asset == nil {
				return fmt.Errorf("%w: no download available for the current host OS + architecture", errNoDownloadAvailable)
			}

			digest, err := publishedDigest(latest, asset.Name)
			if err != nil {
				return err
			}

			var ext string
			if runtime.GOOS == "windows" {
				ext = ".exe"
			}

			return downloadVerified(asset, digest, fmt.Sprintf("launchpad%s", ext))
		},
	}
}
