package releaseutil

import (
	"testing"

	"github.com/Mirantis/launchpad/version"
	"github.com/stretchr/testify/require"
)

// sha256sum writes two spaces between the digest and the name in text mode.
// This is the format the release actually publishes.
func TestDigestForReadsTextModeLines(t *testing.T) {
	checksums := []byte("aaaa  other_asset\nbbbb  launchpad_linux_amd64\n")

	got, err := DigestFor(checksums, "launchpad_linux_amd64")

	require.NoError(t, err)
	require.Equal(t, "bbbb", got)
}

// In binary mode sha256sum writes one space and prefixes the name with "*".
// A checksums file generated that way must still resolve, otherwise the
// download aborts on a file that does list the asset.
func TestDigestForReadsBinaryModeLines(t *testing.T) {
	checksums := []byte("cccc *launchpad_windows_amd64.exe\n")

	got, err := DigestFor(checksums, "launchpad_windows_amd64.exe")

	require.NoError(t, err)
	require.Equal(t, "cccc", got)
}

// Blank lines and lines with no separator are not entries and must not be
// mistaken for one; a trailing newline alone produces such a line.
func TestDigestForSkipsLinesWithoutASeparator(t *testing.T) {
	checksums := []byte("\nnot-an-entry\n\ndddd  launchpad_darwin_arm64\n\n")

	got, err := DigestFor(checksums, "launchpad_darwin_arm64")

	require.NoError(t, err)
	require.Equal(t, "dddd", got)
}

// An asset the checksums file does not list has no published digest, so it
// cannot be verified. That is fatal rather than a reason to write the
// binary unverified.
func TestDigestForFailsWhenAssetIsNotListed(t *testing.T) {
	checksums := []byte("eeee  other_asset\n")

	_, err := DigestFor(checksums, "launchpad_linux_amd64")

	require.ErrorIs(t, err, ErrNoChecksum)
	require.ErrorContains(t, err, "launchpad_linux_amd64")
}

// A prefix of a listed name is not that name: matching on a prefix would
// verify the binary against another asset's digest.
func TestDigestForDoesNotMatchOnAPrefix(t *testing.T) {
	checksums := []byte("ffff  launchpad_linux_amd64_extra\n")

	_, err := DigestFor(checksums, "launchpad_linux_amd64")

	require.ErrorIs(t, err, ErrNoChecksum)
}

func TestChecksumsAssetFindsTheCombinedFile(t *testing.T) {
	release := &version.LaunchpadRelease{
		TagName: "9.9.9",
		Assets: []version.Asset{
			{Name: "launchpad_linux_amd64", URL: "https://github.com/binary"},
			{Name: ChecksumsAssetName, URL: "https://github.com/checksums"},
		},
	}

	got := ChecksumsAsset(release)

	require.NotNil(t, got)
	require.Equal(t, "https://github.com/checksums", got.URL)
}

// A release that publishes no checksums file reports nil, which the caller
// turns into an aborted download rather than an unverified write.
func TestChecksumsAssetReturnsNilWhenAbsent(t *testing.T) {
	release := &version.LaunchpadRelease{
		TagName: "9.9.9",
		Assets:  []version.Asset{{Name: "launchpad_linux_amd64", URL: "https://github.com/binary"}},
	}

	require.Nil(t, ChecksumsAsset(release))
}
