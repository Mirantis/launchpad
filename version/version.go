package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/go-version"
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

// LaunchpadRelease describes a launchpad release
type LaunchpadRelease struct {
	URL     string `json:"html_url"`
	TagName string `json:"tag_name"`

	mutex sync.Mutex `json:"-"`
}

func (l *LaunchpadRelease) UpgradeMessage() string {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.TagName == "" {
		return ""
	}

	current, err := version.NewVersion(Version)
	remote, err := version.NewVersion(l.TagName)
	if err != nil {
		return "" // ignore invalid versions
	}
	if current.LessThan(remote) {
		return fmt.Sprintf("A new version (%s) of `launchpad` is available. Please visit %s to upgrade the tool.", l.TagName, l.URL)
	}
	return ""
}

// GetLatest will populate the release information from Github launchpad repository latest releases
func (l *LaunchpadRelease) GetLatest() {
	if !IsProduction() {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	client := &http.Client{
		Timeout: time.Second * 2,
	}

	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo))
	if err != nil {
		return // ignore connection errors
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		return // ignore backend failures
	}
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return // ignore reading errors
	}

	json.Unmarshal(body, &l)
}
