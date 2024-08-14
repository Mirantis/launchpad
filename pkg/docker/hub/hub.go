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

const (
	RegistryDockerHub = "hub.docker.com"
	RegistryMirantis  = "registry.mirantis.com"
	RegistryMsrCi     = "msr.ci.mirantis.com"

	DockerHubImageURLPattern        = "https://hub.docker.com/v2/repositories/%s/%s/tags"
	MirantisRegistryImageURLPattern = "https://registry.mirantis.com/api/v0/repositories/%s/%s/tags"
	MsrCIImageURLPattern            = "https://msr.ci.mirantis.com/api/v0/repositories/%s/%s/tags"
)

var (
	errQueryFailed    = fmt.Errorf("latest version query failed")
	errUnkownRegistry = fmt.Errorf("unknown registry")
)

// LatestTag returns the latest tag name from a public docker hub repository.
// If pre is true, also prereleases are considered.
func LatestTag(registry, org, image string, pre bool) (string, error) {
	var pattern string
	var parser func([]byte, bool) (string, error)

	switch registry {
	case RegistryDockerHub:
		pattern = DockerHubImageURLPattern
		parser = TagParserDockerHub
	case RegistryMirantis:
		pattern = MirantisRegistryImageURLPattern
		parser = TagParserMSR
	case RegistryMsrCi:
		pattern = MsrCIImageURLPattern
		parser = TagParserMSR
	default:
		return "", fmt.Errorf("%w %s", errUnkownRegistry, registry)
	}

	url := fmt.Sprintf(pattern, org, image)
	client := http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("%w: request build failed %w", errQueryFailed, err)
	}

	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: client failure '%s' %w", errQueryFailed, url, err)
	}

	if res == nil {
		return "", errQueryFailed
	}

	if res.Body != nil {
		defer res.Body.Close()
	}
	if res.StatusCode > 299 || res.StatusCode < http.StatusOK {
		return "", fmt.Errorf("%w: failed response status %d: %s", errQueryFailed, res.StatusCode, url)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("%w: read response body: %w", errQueryFailed, err)
	}

	return parser(body, pre)
}

func TagParserDockerHub(r []byte, pre bool) (string, error) {
	var taglist struct {
		Count   int `json:"count"`
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.Unmarshal(r, &taglist); err != nil {
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

func TagParserMSR(r []byte, pre bool) (string, error) {
	var taglist []struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(r, &taglist); err != nil {
		return "", fmt.Errorf("%w: unmarshal response: %w", errQueryFailed, err)
	}

	var tags []*version.Version
	for _, t := range taglist {
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
