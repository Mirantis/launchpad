package configurer

import (
	"crypto/sha256"
	"encoding/hex"
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

	// errInstallerChecksumMismatch aborts when an operator-supplied digest
	// does not match what was fetched.
	errInstallerChecksumMismatch = errors.New("container runtime install script does not match the configured checksum")
)

// GetInstaller fetches the container runtime install script and returns the
// path of the local copy.
//
// checksum is the operator-supplied SHA-256 of the script, hex encoded, or
// empty. It is empty by default because Mirantis publishes nothing to pin
// against: as measured for PRODENG-3924, https://get.mirantis.com/install.ps1
// returns 200 while install.ps1.sha256, install.ps1.asc, install.ps1.sig and
// checksums.sha256 all return 404. There is no upstream integrity anchor to
// verify against, so the default protection is the https transport and the
// pin exists for operators who mirror the script themselves and know its
// digest.
//
// The cache key includes the checksum so that two hosts configured with the
// same URL and different pins cannot share a verification result.
func GetInstaller(source, checksum string) (string, error) {
	cacheKey := source + "\x00" + checksum

	path, ok := downloadedInstallers[cacheKey]
	if ok {
		return path, nil
	}

	path, getErr := downloadInstaller(source, checksum)
	if getErr != nil {
		return "", fmt.Errorf("%w, installer download failed; %s", ErrInstallerDownloadFailed, getErr.Error())
	}
	downloadedInstallers[cacheKey] = path
	return path, nil
}

// verifyInstallerChecksum compares the fetched script against the configured
// digest.
//
// A mismatch is fatal and there is no override. An operator who sets a pin
// has asked for the fetch to be verified; ignoring the pin with a warning
// would be worse than not offering one, because the configuration would
// claim a guarantee the run does not provide.
func verifyInstallerChecksum(script, checksum string) error {
	if checksum == "" {
		return nil
	}

	sum := sha256.Sum256([]byte(script))
	got := hex.EncodeToString(sum[:])

	if !strings.EqualFold(got, strings.TrimSpace(checksum)) {
		return fmt.Errorf("%w: expected %s, got %s", errInstallerChecksumMismatch, checksum, got)
	}

	return nil
}

// downloadInstaller fetches the script, verifies it, and stages it locally.
//
// Verification happens before the local copy exists, so a rejected script
// leaves nothing on disk for a later step to pick up.
func downloadInstaller(path, checksum string) (string, error) {
	winScript, err := getScript(path)
	if err != nil {
		return "", fmt.Errorf("failed to get Windows installer script: %w", err)
	}

	if err := verifyInstallerChecksum(winScript, checksum); err != nil {
		return "", err
	}

	staged, err := os.CreateTemp("", "installerWindows")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file for windows installer script: %w", err)
	}
	defer staged.Close()

	_, err = staged.WriteString(winScript)
	if err != nil {
		return "", fmt.Errorf("failed to write to temporary file for windows installer script: %w", err)
	}

	return staged.Name(), nil
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

func downloadFile(url string) (string, error) {
	log.Infof("downloading container runtime install script from %s", url)
	// The URL is operator-supplied by design (spec.mcr.installURLWindows);
	// getScript has already rejected every scheme but https.
	resp, err := http.Get(url) //nolint:gosec // G107: taint is inherent to the feature; scheme is validated in getScript
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
