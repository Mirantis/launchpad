package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

var (
	// Version of the product, is set during the build.
	Version = "0.0.0"
	// GitCommit is set during the build.
	GitCommit = "HEAD"
	// Environment of the product, is set during the build.
	Environment = "production"

	// GitHubRepo for the upgrade check.
	GitHubRepo = "Mirantis/launchpad"
)

// IsProduction tells if running production build.
func IsProduction() bool {
	return Environment == "production"
}

// Asset describes a github release asset.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// IsForHost returns true if the asset is for the current host os+arch.
func (a *Asset) IsForHost() bool {
	if strings.HasSuffix(a.Name, ".sha256") {
		return false
	}

	parts := strings.Split(strings.TrimSuffix(a.Name, ".exe"), "_")
	if len(parts) < 3 {
		return false
	}
	return parts[1] == runtime.GOOS && parts[2] == runtime.GOARCH
}

// LaunchpadRelease describes a launchpad release.
type LaunchpadRelease struct {
	URL     string  `json:"html_url"`
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// IsNewer returns true if the LaunchpadRelease instance is newer than the current version.
func (l *LaunchpadRelease) IsNewer() bool {
	current, _ := version.NewVersion(Version)
	remote, err := version.NewVersion(l.TagName)
	if err != nil {
		return false // ignore invalid versions
	}

	return current.LessThan(remote)
}

// AssetForHost returns a download asset for the current host OS+ARCH if available.
func (l *LaunchpadRelease) AssetForHost() *Asset {
	for _, a := range l.Assets {
		if a.IsForHost() {
			return &a
		}
	}
	return nil
}

// ChecksumsAssetName is the name of the release asset holding the SHA-256
// digests of every other asset.
//
// The release publishes one combined file in sha256sum format ("<hex>  <name>"
// per line), not a per-asset <name>.sha256 sibling. IsForHost filters any
// name ending in .sha256, which excludes this file from the download
// candidates as well.
const ChecksumsAssetName = "checksums.sha256"

// ChecksumsAsset returns the release's combined checksums asset, or nil when
// the release does not publish one.
func (l *LaunchpadRelease) ChecksumsAsset() *Asset {
	for i := range l.Assets {
		if l.Assets[i].Name == ChecksumsAssetName {
			return &l.Assets[i]
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

type tag struct {
	Name string `json:"name"`
}

// ErrNoRelease is returned when the latest release cannot be determined. The
// previous contract was a nil return, which every caller had to remember to
// check and two of three did not.
var ErrNoRelease = errors.New("could not determine the latest launchpad release")

// githubAPIBase is a variable rather than a constant so tests can point the
// lookup at a local server and exercise the failure paths, which are
// otherwise reachable only by exhausting the real API quota.
var githubAPIBase = "https://api.github.com"

func latestTag(timeout time.Duration) (string, error) {
	client := &http.Client{
		Timeout: timeout,
	}

	baseMsg := "getting launchpad tag list"
	log.Debug(baseMsg)
	resp, err := client.Get(fmt.Sprintf("%s/repos/%s/tags?per_page=20&page=1", githubAPIBase, GitHubRepo))
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrNoRelease, baseMsg, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// The unauthenticated api.github.com quota is 60/hour and shared by
		// every process on the host, so this is the common failure and the
		// message needs to say which it was.
		return "", fmt.Errorf("%w: %s returned http %d", ErrNoRelease, baseMsg, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %s failed to read body: %w", ErrNoRelease, baseMsg, err)
	}

	var tags []tag
	if err := json.Unmarshal(body, &tags); err != nil {
		return "", fmt.Errorf("%w: %s failed to unmarshal JSON: %w", ErrNoRelease, baseMsg, err)
	}

	return pickLatest(tags, baseMsg)
}

// pickLatest returns the newest parseable tag name. It is separate from the
// HTTP plumbing so the selection logic can be tested directly: the previous
// implementation filled a len(tags) slice by an index that advanced only on a
// successful parse, leaving trailing nil entries that sort.Sort dereferences
// in Less. One unparseable name in an otherwise successful response was enough
// to panic, with no HTTP failure involved.
func pickLatest(tags []tag, baseMsg string) (string, error) {
	versions := make([]*version.Version, 0, len(tags))
	for _, v := range tags {
		parsed, err := version.NewVersion(v.Name)
		if err != nil {
			log.Debugf("%s: ignoring unparseable tag %q", baseMsg, v.Name)
			continue
		}
		versions = append(versions, parsed)
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("%w: %s returned no parseable tags", ErrNoRelease, baseMsg)
	}
	sort.Sort(version.Collection(versions))

	if strings.Contains(Version, "-") {
		// Current is pre, assume pre is accepted as latest
		return versions[len(versions)-1].Original(), nil
	}

	for i := len(versions) - 1; i >= 0; i-- {
		if !strings.Contains(versions[i].Original(), "-") {
			return versions[i].Original(), nil
		}
	}

	return "", fmt.Errorf("%w: %s returned only prerelease tags", ErrNoRelease, baseMsg)
}

// GetLatest returns a LaunchpadRelease instance for the latest release, or an
// error describing why it could not be determined. It previously returned a
// bare nil on every failure, an unexpressed contract that two of its three
// callers did not honour.
func GetLatest(timeout time.Duration) (*LaunchpadRelease, error) {
	tag, err := latestTag(timeout)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: timeout,
	}

	baseMsg := fmt.Sprintf("getting launchpad release information for version %s", tag)
	log.Debug(baseMsg)
	resp, err := client.Get(fmt.Sprintf("%s/repos/%s/releases/tags/%s", githubAPIBase, GitHubRepo, tag))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrNoRelease, baseMsg, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s returned http %d", ErrNoRelease, baseMsg, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s failed to read body: %w", ErrNoRelease, baseMsg, err)
	}

	l := &LaunchpadRelease{}
	if err := json.Unmarshal(body, l); err != nil {
		return nil, fmt.Errorf("%w: %s failed to unmarshal JSON: %w", ErrNoRelease, baseMsg, err)
	}

	return l, nil
}

// GetUpgrade will return a LaunchpadRelease instance for the latest release if it's newer than the current version.
// A nil return means no upgrade is available; the lookup failing is logged and
// treated the same way, because an upgrade check must never block a command.
func GetUpgrade() *LaunchpadRelease {
	log.Debugf("checking for a launchpad upgrade")
	l, err := GetLatest(time.Second * 2)
	if err != nil {
		log.Debugf("upgrade check failed: %s", err.Error())
		return nil
	}

	if l.IsNewer() {
		return l
	}

	return nil
}
