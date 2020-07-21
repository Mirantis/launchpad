package util

import (
	"fmt"
	"net/url"
	"strings"
)

// ResolveURL resolves a serverURL into url.URL
func ResolveURL(serverURL string) (*url.URL, error) {
	if !strings.HasPrefix(serverURL, "https://") {
		serverURL = fmt.Sprintf("https://%s", serverURL)
	}
	url, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	return url, nil
}
