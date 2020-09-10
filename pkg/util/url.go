package util

import (
	"fmt"
	"net/url"
	"path"
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

// GetFileFromURL returns the last element (filename) from an URL's Path
func GetFileFromURL(serverURL string) (string, error) {
	u, err := ResolveURL(serverURL)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}
