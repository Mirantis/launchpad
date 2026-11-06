// Package releaseutil reads the integrity metadata a launchpad release
// publishes alongside its binaries, so a downloaded asset can be checked
// against a digest the release itself vouches for.
package releaseutil

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Mirantis/launchpad/version"
)

// ChecksumsAssetName is the name of the release asset holding the SHA-256
// digests of every other asset.
//
// The release publishes one combined file in sha256sum format ("<hex>  <name>"
// per line), not a per-asset <name>.sha256 sibling. Asset.IsForHost filters
// any name ending in .sha256, which excludes this file from the download
// candidates as well.
const ChecksumsAssetName = "checksums.sha256"

// ChecksumsAsset returns the release's combined checksums asset, or nil when
// the release does not publish one.
func ChecksumsAsset(release *version.LaunchpadRelease) *version.Asset {
	for i := range release.Assets {
		if release.Assets[i].Name == ChecksumsAssetName {
			return &release.Assets[i]
		}
	}

	return nil
}

// ErrNoChecksum is returned when the checksums file does not list the asset
// being downloaded. It is fatal: an asset with no published digest cannot be
// verified, and writing an unverified executable is the thing this avoids.
var ErrNoChecksum = errors.New("no published checksum for the release asset")

// DigestFor returns the hex-encoded SHA-256 the checksums file lists for the
// named asset.
func DigestFor(checksums []byte, name string) (string, error) {
	for _, line := range strings.Split(string(checksums), "\n") {
		digest, listed, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found {
			continue
		}

		// sha256sum writes two spaces, or a space and a "*" for binary
		// mode; either way the name is what remains after trimming.
		if strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(listed), "*")) == name {
			return digest, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrNoChecksum, name)
}
