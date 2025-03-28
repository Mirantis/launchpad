package version

import (
	"encoding/json"
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

type tag struct {
	Name string `json:"name"`
}

func latestTag(timeout time.Duration) string {
	client := &http.Client{
		Timeout: timeout,
	}

	baseMsg := "getting launchpad tag list"
	log.Debug(baseMsg)
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=20&page=1", GitHubRepo))
	if err != nil {
		log.Debugf("%s failed: %s", baseMsg, err.Error())
		return "" // ignore connection errors
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		log.Debugf("%s returned http %d", baseMsg, resp.StatusCode)
		return "" // ignore backend failures
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Debugf("%s failed to read body: %s", baseMsg, err.Error())
		return "" // ignore reading errors
	}

	var tags []tag
	err = json.Unmarshal(body, &tags)
	if err != nil {
		log.Debugf("%s failed to unmarshal JSON: %s", baseMsg, err.Error())
		return ""
	}

	versions := make([]*version.Version, len(tags))
	idx := 0
	for _, v := range tags {
		version, err := version.NewVersion(v.Name)
		if err == nil {
			versions[idx] = version
			idx++
		}
	}
	sort.Sort(version.Collection(versions))

	if strings.Contains(Version, "-") {
		// Current is pre, assume pre is accepted as latest
		return versions[len(versions)-1].Original()
	}

	for i := len(versions) - 1; i >= 0; i-- {
		if !strings.Contains(versions[i].Original(), "-") {
			return versions[i].Original()
		}
	}

	return ""
}

// GetLatest returns a LaunchpadRelease instance for the latest release.
func GetLatest(timeout time.Duration) *LaunchpadRelease {
	tag := latestTag(timeout)
	if tag == "" {
		return nil
	}

	client := &http.Client{
		Timeout: timeout,
	}

	baseMsg := fmt.Sprintf("getting launchpad release information for version %s", tag)
	log.Debug(baseMsg)
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", GitHubRepo, tag))
	if err != nil {
		log.Debugf("%s failed: %s", baseMsg, err.Error())
		return nil // ignore connection errors
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		log.Debugf("%s returned http %d", baseMsg, resp.StatusCode)
		return nil // ignore backend failures
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Debugf("%s failed to read body: %s", baseMsg, err.Error())
		return nil // ignore reading errors
	}

	l := &LaunchpadRelease{}
	err = json.Unmarshal(body, l)
	if err != nil {
		log.Debugf("%s failed to unmarshal JSON: %s", baseMsg, err.Error())
		return nil
	}

	return l
}

// GetUpgrade will return a LaunchpadRelease instance for the latest release if it's newer than the current version.
func GetUpgrade() *LaunchpadRelease {
	log.Debugf("checking for a launchpad upgrade")
	l := GetLatest(time.Second * 2)
	if l == nil {
		return nil
	}

	if l.IsNewer() {
		return l
	}

	return nil
}
