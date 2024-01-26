package hub

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

type tagListResponse struct {
	Count   int `json:"count"`
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

var errQueryFailed = fmt.Errorf("latest version query failed, you can try running with --disable-upgrade-check")

// LatestTag returns the latest tag name from a public docker hub repository.
// If pre is true, also prereleases are considered.
func LatestTag(org, image string, pre bool) (string, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags", org, image)
	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errQueryFailed, err)
	}

	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errQueryFailed, err)
	}

	if res == nil {
		return "", errQueryFailed
	}

	if res.Body != nil {
		defer res.Body.Close()
	}
	if res.StatusCode > 299 || res.StatusCode < http.StatusOK {
		return "", fmt.Errorf("%w: response status %d", errQueryFailed, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("%w: read response body: %w", errQueryFailed, err)
	}
	var taglist tagListResponse

	if err := json.Unmarshal(body, &taglist); err != nil {
		return "", fmt.Errorf("%w: unmarshal response: %w", errQueryFailed, err)
	}

	var tags []*version.Version
	for _, t := range taglist.Results {
		if !pre && strings.Contains(t.Name, "-") {
			continue
		}

		if v, err := version.NewVersion(t.Name); err == nil {
			tags = append(tags, v)
		}
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("%w: no tags received", errQueryFailed)
	}
	sort.Sort(version.Collection(tags))
	return tags[len(tags)-1].String(), nil
}
