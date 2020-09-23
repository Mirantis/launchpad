package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
)

var (
	// Version of the product, is set during the build
	Version = "0.0.0"
	// GitCommit is set during the build
	GitCommit = "HEAD"
	// Environment of the product, is set during the build
	Environment = "development"

	// GitHubRepo for the upgrade check
	GitHubRepo = "Mirantis/launchpad"
)

// IsProduction tells if running production build
func IsProduction() bool {
	return Environment == "production"
}

// Asset describes a github release asset
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// IsForHost returns true if the asset is for the current host os+arch
func (a *Asset) IsForHost() bool {
	if strings.HasSuffix(a.Name, ".sha256") {
		return false
	}

	os := runtime.GOOS
	if os == "windows" {
		os = "win"
	}

	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}

	parts := strings.Split(strings.TrimSuffix(a.Name, ".exe"), "-")
	return parts[1] == os && parts[2] == arch
}

// LaunchpadRelease describes a launchpad release
type LaunchpadRelease struct {
	URL     string  `json:"html_url"`
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// IsNewer returns true if the LaunchpadRelease instance is newer than the current version
func (l *LaunchpadRelease) IsNewer() bool {
	current, _ := version.NewVersion(Version)
	remote, err := version.NewVersion(l.TagName)
	if err != nil {
		return false // ignore invalid versions
	}

	return current.LessThan(remote)
}

// AssetForHost returns a download asset for the current host OS+ARCH if available
func (l *LaunchpadRelease) AssetForHost() *Asset {
	for _, a := range l.Assets {
		if a.IsForHost() {
			return &a
		}
	}
	return nil
}

// GetLatest returns a LaunchpadRelease instance for the latest release
func GetLatest(timeout time.Duration) *LaunchpadRelease {
	client := &http.Client{
		Timeout: timeout,
	}

	baseMsg := "getting launchpad release information"
	log.Debugf(baseMsg)
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo))
	if err != nil {
		log.Debugf("%s failed: %s", baseMsg, err.Error())
		return nil // ignore connection errors
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		log.Debugf("%s returned http %d", baseMsg, resp.StatusCode)
		return nil // ignore backend failures
	}
	body, readErr := ioutil.ReadAll(resp.Body)
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

	log.Debugf("%s returned: %+v", baseMsg, *l)
	return l
}

// GetUpgrade will return a LaunchpadRelease instance for the latest release if it's newer than the current version
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
