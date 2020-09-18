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

// GetLatest returns a LaunchpadRelease instance for the latest release
func GetLatest() *LaunchpadRelease {
	client := &http.Client{
		Timeout: time.Second * 2,
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
	l := GetLatest()
	if l == nil {
		return nil
	}

	current, err := version.NewVersion(Version)
	remote, err := version.NewVersion(l.TagName)
	if err != nil {
		log.Debugf("checking for a launchpad upgrade invalid version: %s", err.Error())
		return nil // ignore invalid versions
	}

	if current.LessThan(remote) {
		log.Debugf("checking for a launchpad upgrade found a new version %s", remote.String())
		return l
	}

	return nil
}
