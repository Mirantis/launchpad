package phase

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	github.com/Mirantis/launchpad/pkg/phase"
	github.com/Mirantis/launchpad/pkg/product/mke/api"
	github.com/Mirantis/launchpad/pkg/util/fileutil"
	log "github.com/sirupsen/logrus"
)

// DownloadInstaller phase implementation does all the prep work we need for the hosts.
type DownloadInstaller struct {
	phase.Analytics
	phase.BasicPhase

	linuxPath string
	winPath   string
}

// Title for the phase.
func (p *DownloadInstaller) Title() string {
	return "Download Mirantis Container Runtime installer"
}

// Run does all the prep work on the hosts in parallel.
func (p *DownloadInstaller) Run() error {
	linuxScript, err := p.getScript(p.Config.Spec.MCR.InstallURLLinux)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp("", "installerLinux")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	_, err = f.WriteString(linuxScript)
	if err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	p.linuxPath = f.Name()

	if p.Config.Spec.Hosts.Count(func(h *api.Host) bool { return h.IsWindows() }) > 0 {
		winScript, err := p.getScript(p.Config.Spec.MCR.InstallURLWindows)
		if err != nil {
			return fmt.Errorf("failed to get Windows installer script: %w", err)
		}
		f, err := os.CreateTemp("", "installerWindows")
		if err != nil {
			return fmt.Errorf("failed to create temporary file for windows installer script: %w", err)
		}

		_, err = f.WriteString(winScript)
		if err != nil {
			return fmt.Errorf("failed to write to temporary file for windows installer script: %w", err)
		}
		p.winPath = f.Name()
	}

	for _, h := range p.Config.Spec.Hosts {
		if h.IsWindows() {
			h.Metadata.MCRInstallScript = p.winPath
		} else {
			h.Metadata.MCRInstallScript = p.linuxPath
		}
	}

	return nil
}

func (p *DownloadInstaller) parseURL(uri string) (*url.URL, error) {
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

func (p *DownloadInstaller) getScript(uri string) (string, error) {
	u, err := p.parseURL(uri)
	if err != nil {
		return "", err
	}

	var data string

	if u.Scheme == "file" {
		data, err = p.readFile(u.Path)
	} else {
		data, err = p.downloadFile(uri)
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

func (p *DownloadInstaller) downloadFile(url string) (string, error) {
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

func (p *DownloadInstaller) readFile(path string) (string, error) {
	log.Infof("reading container runtime install script from %s", path)

	data, err := fileutil.LoadExternalFile(path)
	return string(data), err
}

// CleanUp removes the temporary files from local filesystem.
func (p *DownloadInstaller) CleanUp() {
	if p.winPath != "" {
		removeIfExist(p.winPath)
	}
	if p.linuxPath != "" {
		removeIfExist(p.linuxPath)
	}
}

func removeIfExist(path string) {
	_, err := os.Stat(path)
	if err == nil {
		os.Remove(path)
	}
}
