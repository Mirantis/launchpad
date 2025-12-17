package configurer

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Mirantis/launchpad/pkg/util/fileutil"
	log "github.com/sirupsen/logrus"
)

var (
	downloadedInstallers       = map[string]string{} // Global list of downloaded installers, to prevent repetition.
	ErrInstallerDownloadFailed = errors.New("could not download installer")
)

func GetInstaller(source string) (string, error) {
	path, ok := downloadedInstallers[source]
	if ok {
		return path, nil
	}

	if path == "" {
		return "", fmt.Errorf("%w; skipping failed installer download", ErrInstallerDownloadFailed)
	}

	path, getErr := downloadInstaller(source)
	if getErr != nil {
		return "", fmt.Errorf("%w, installer download failed; %s", ErrInstallerDownloadFailed, getErr.Error())
	}
	downloadedInstallers[source] = path
	return path, nil
}

// Run does all the prep work on the hosts in parallel.
func downloadInstaller(path string) (string, error) {
	winScript, err := getScript(path)
	if err != nil {
		return "", fmt.Errorf("failed to get Windows installer script: %w", err)
	}
	f, err := os.CreateTemp("", "installerWindows")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file for windows installer script: %w", err)
	}

	_, err = f.WriteString(winScript)
	if err != nil {
		return "", fmt.Errorf("failed to write to temporary file for windows installer script: %w", err)
	}

	return f.Name(), nil
}

func parseURL(uri string) (*url.URL, error) {
	if !strings.Contains(uri, "://") {
		return &url.URL{Path: uri, Scheme: "file"}, nil
	}

	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installer URL: %w", err)
	}
	return u, nil
}

var errInvalidScript = fmt.Errorf("invalid container runtime install script")

func getScript(uri string) (string, error) {
	u, err := parseURL(uri)
	if err != nil {
		return "", err
	}

	var data string

	if u.Scheme == "file" {
		data, err = readFile(u.Path)
	} else {
		data, err = downloadFile(uri)
	}

	log.Debugf("read %d bytes from %s", len(data), uri)

	if err != nil {
		return "", err
	}

	if len(data) < 10 {
		// cant fit an installer into that!
		return "", fmt.Errorf("%w: script is too short", errInvalidScript)
	}

	if !strings.HasPrefix(data, "#") {
		log.Warnf("possibly invalid container runtime install script in %s", uri)
	}

	return data, nil
}

func downloadFile(url string) (string, error) {
	log.Infof("downloading container runtime install script from %s", url)
	resp, err := http.Get(url) //nolint:gosec // "G107: Url provided to HTTP request as taint input" -- user-provided URL is ok here
	if err != nil {
		return "", fmt.Errorf("failed to download container runtime install script: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(body), nil
}

func readFile(path string) (string, error) {
	log.Infof("reading container runtime install script from %s", path)

	data, err := fileutil.LoadExternalFile(path)
	return string(data), err
}
