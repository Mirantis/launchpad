package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

// LaunchpadRelease describes a launchpad release
type LaunchpadRelease struct {
	URL     string `json:"html_url"`
	TagName string `json:"tag_name"`
}

// UpgradeMessage returns a friendly upgrade message (after the GetLatest has released the mutex) or an empty string if there is no upgrade available
func (l *LaunchpadRelease) UpgradeMessage() string {
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
func GetUpgrade() *LaunchpadRelease {
	client := &http.Client{
		Timeout: time.Second * 2,
	}

	log.Debugf("checking for launchpad upgrade")
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GitHubRepo))
	if err != nil {
		log.Debugf("checking for launchpad upgrade failed: %s", err.Error())
		return nil // ignore connection errors
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != 200 {
		log.Debugf("checking for launchpad upgrade returned http %d", resp.StatusCode)
		return nil // ignore backend failures
	}
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Debugf("checking for launchpad upgrade failed to read body: %s", err.Error())
		return nil // ignore reading errors
	}

	l := &LaunchpadRelease{}
	err = json.Unmarshal(body, l)
	if err != nil {
		log.Debugf("checking for launchpad upgrade failed to unmarshal JSON: %s", err.Error())
		return nil
	}

	log.Debugf("checking for launchpad upgrade returned: %+v", *l)

	current, err := version.NewVersion(Version)
	remote, err := version.NewVersion(l.TagName)
	if err != nil {
		log.Debugf("checking for launchpad upgrade invalid version: %s", err.Error())
		return nil // ignore invalid versions
	}

	if current.LessThan(remote) {
		log.Debugf("checking for launchpad upgrade found a new version %s", remote.String())
		return l
	}

	return nil
}
