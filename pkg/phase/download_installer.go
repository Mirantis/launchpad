package phase

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// DownloadInstaller phase implementation does all the prep work we need for the hosts
type DownloadInstaller struct {
	Analytics
}

// Title for the phase
func (p *DownloadInstaller) Title() string {
	return "Download installer"
}

// Run does all the prep work on the hosts in parallel
func (p *DownloadInstaller) Run(config *api.ClusterConfig) error {
	linuxScript, err := p.getScript(config.Spec.Engine.InstallURLLinux)
	if err != nil {
		return err
	}

	var winScript string

	wincount := config.Spec.Hosts.Count(func(h *api.Host) bool { return h.IsWindows() })
	if wincount > 0 {
		winScript, err = p.getScript(config.Spec.Engine.InstallURLWindows)
		if err != nil {
			return err
		}
	}

	for _, h := range config.Spec.Hosts {
		if h.IsWindows() {
			h.Metadata.EngineInstallScript = &winScript
		} else {
			h.Metadata.EngineInstallScript = &linuxScript
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
		return "", fmt.Errorf("invalid engine install script in %s", uri)
	}

	if !strings.HasPrefix(data, "#") {
		log.Warnf("possibly invalid engine install script in %s", uri)
	}

	return data, nil
}

func (p *DownloadInstaller) downloadFile(url string) (string, error) {
	log.Infof("downloading engine install script from %s", url)
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
	log.Infof("reading engine install script from %s", path)

	data, err := util.LoadExternalFile(path)
	return string(data), err
}
