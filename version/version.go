package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

type release struct {
	Name    string `json:"name"`
	URL     string `json:"html_url"`
	TagName string `json:"tag_name"`
}

// CheckForUpgrade detects if newer version is available
func CheckForUpgrade() {
	if !IsProduction() {
		return // do not check on dev builds
	}
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

	latest := release{}
	err = json.Unmarshal(body, &latest)
	if err != nil {
		return // ignore json parsing errors
	}
	current, err := version.NewVersion(Version)
	remote, err := version.NewVersion(latest.TagName)
	if err != nil {
		return // ignore invalid versions
	}
	if current.LessThan(remote) {
		fmt.Println("")
		fmt.Println(fmt.Sprintf("New version (%s) of the `launchpad` is available. Please visit %s to upgrade the tool.", latest.Name, latest.URL))""

	}

}
