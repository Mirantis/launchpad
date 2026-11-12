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

	// errInsecureInstallerURL rejects a transport that cannot authenticate
	// the server. The bytes fetched here are uploaded to every Windows host
	// and executed with -ExecutionPolicy Bypass, so plain http would let
	// anyone on the path choose what runs as Administrator.
	errInsecureInstallerURL = errors.New("container runtime install script must be fetched over https or read from a local file")
)

func GetInstaller(source string) (string, error) {
	path, ok := downloadedInstallers[source]
	if ok {
		return path, nil
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

// getScript reads the install script from a local path or fetches it over
// https.
//
// Any scheme other than https is rejected. The previous behaviour passed
// whatever url.ParseRequestURI accepted straight to http.Get, so a config
// naming an http:// URL fetched an executable over a transport that cannot
// authenticate the server, and the bytes were then run on every Windows
// host with -ExecutionPolicy Bypass. A local file remains allowed, which is
// how an air-gapped install supplies its own script.
func getScript(uri string) (string, error) {
	parsed, err := parseURL(uri)
	if err != nil {
		return "", err
	}

	var data string

	switch parsed.Scheme {
	case "file":
		data, err = readFile(parsed.Path)
	case "https":
		data, err = downloadFile(uri)
	default:
		return "", fmt.Errorf("%w: %s uses scheme %q", errInsecureInstallerURL, uri, parsed.Scheme)
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

// errInstallerHTTPStatus aborts on any non-200 response. Without this, a 404
// page was stored as the install script and uploaded to the hosts: the body
// of an error response is still a body, and the only thing standing between
// it and execution was the 10-byte length check.
var errInstallerHTTPStatus = errors.New("unexpected HTTP status fetching container runtime install script")

// installerClient refuses to follow a redirect off https.
//
// Validating only the configured URL is not enough: the server it names
// chooses where the request goes next, and the default client follows an
// https-to-http redirect without complaint. The final response would carry
// a 200 and the script would run on every Windows host having crossed a
// transport that cannot authenticate the server, which is the thing the
// scheme check exists to prevent. It is a variable so tests can supply a
// client that trusts httptest's CA.
var installerClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}

		if req.URL.Scheme != "https" {
			return fmt.Errorf("%w: redirected to %s", errInsecureInstallerURL, req.URL.Redacted())
		}

		return nil
	},
}

func downloadFile(url string) (string, error) {
	log.Infof("downloading container runtime install script from %s", url)

	// The URL is operator-supplied by design (spec.mcr.installURLWindows);
	// getScript rejects every scheme but https, and installerClient rejects
	// a redirect that leaves it.
	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx // no context is threaded through this call path
	if err != nil {
		return "", fmt.Errorf("failed to build the install script request: %w", err)
	}

	resp, err := installerClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download container runtime install script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s returned %s", errInstallerHTTPStatus, url, resp.Status)
	}

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
