package phase

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// DownloadInstaller phase implementation does all the prep work we need for the hosts
type DownloadInstaller struct {
	phase.Analytics
	phase.BasicPhase

	linuxPath string
	winPath   string
}

// Title for the phase
func (p *DownloadInstaller) Title() string {
	return "Download Mirantis Container Runtime installer"
}

// Run does all the prep work on the hosts in parallel
func (p *DownloadInstaller) Run() error {
	linuxScript, err := p.getScript(p.Config.Spec.MCR.InstallURLLinux)
	if err != nil {
		return err
	}
	f, err := ioutil.TempFile("", "installerLinux")
	if err != nil {
		return err
	}

	_, err = f.WriteString(linuxScript)
	if err != nil {
		return err
	}
	p.linuxPath = f.Name()

	if p.Config.Spec.Hosts.Count(func(h *api.Host) bool { return h.IsWindows() }) > 0 {
		winScript, err := p.getScript(p.Config.Spec.MCR.InstallURLWindows)
		if err != nil {
			return err
		}
		f, err := ioutil.TempFile("", "installerWindows")
		if err != nil {
			return err
		}

		_, err = f.WriteString(winScript)
		if err != nil {
			return err
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

	return url.ParseRequestURI(uri)
}

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
		return "", fmt.Errorf("invalid container runtime install script in %s", uri)
	}

	if !strings.HasPrefix(data, "#") {
		log.Warnf("possibly invalid container runtime install script in %s", uri)
	}

	return data, nil
}

func (p *DownloadInstaller) downloadFile(url string) (string, error) {
	log.Infof("downloading container runtime install script from %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (p *DownloadInstaller) readFile(path string) (string, error) {
	log.Infof("reading container runtime install script from %s", path)

	data, err := util.LoadExternalFile(path)
	return string(data), err
}

// CleanUp removes the temporary files from local filesystem
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
