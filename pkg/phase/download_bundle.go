package phase

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/ucp"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/mitchellh/go-homedir"

	log "github.com/sirupsen/logrus"
)

// DownloadBundle phase downloads the client bundle to local storage
type DownloadBundle struct {
	Analytics
	BasicPhase
	Username string
	Password string
}

// Title for the phase
func (p *DownloadBundle) Title() string {
	return "Download Client Bundle"
}

// Run collect all the facts from hosts in parallel
func (p *DownloadBundle) Run() error {
	m := p.config.Spec.Managers()[0]

	tlsConfig, err := ucp.GetTLSConfigFrom(m, p.config.Spec.Ucp.ImageRepo, p.config.Spec.Ucp.Version)
	if err != nil {
		return fmt.Errorf("error getting TLS config: %w", err)
	}

	url, err := p.config.Spec.UcpURL()
	if err != nil {
		return err
	}

	bundle, err := ucp.GetClientBundle(url, tlsConfig, p.Username, p.Password)
	if err != nil {
		return fmt.Errorf("failed to download admin bundle: %s", err)
	}

	bundleDir, err := p.getBundleDir(p.config.Metadata.Name, p.Username)
	if err != nil {
		return err
	}
	err = p.writeBundle(bundleDir, bundle)
	if err != nil {
		return fmt.Errorf("failed to write admin bundle: %s", err)
	}

	return nil
}

func (p *DownloadBundle) getBundleDir(clusterName, username string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return path.Join(home, constant.StateBaseDir, "cluster", clusterName, "bundle", username), nil
}

func (p *DownloadBundle) writeBundle(bundleDir string, bundle *zip.Reader) error {
	if err := util.EnsureDir(bundleDir); err != nil {
		return fmt.Errorf("error while creating directory: %w", err)
	}
	log.Debugf("Writing out bundle to %s", bundleDir)
	for _, zf := range bundle.File {
		src, err := zf.Open()
		if err != nil {
			return err
		}
		defer src.Close()
		var data []byte
		data, err = ioutil.ReadAll(src)
		if err != nil {
			return err
		}
		mode := int64(0644)
		if strings.Contains(zf.Name, "key.pem") {
			mode = 0600
		}

		// UCP bundle will contain folders as well as files, if folder exists fd will not be empty
		dir, _ := filepath.Split(zf.Name)
		if dir != "" {
			if err := os.MkdirAll(filepath.Join(bundleDir, dir), 0700); err != nil {
				return err
			}
		}

		err = ioutil.WriteFile(filepath.Join(bundleDir, zf.Name), data, os.FileMode(mode))
		if err != nil {
			return err
		}
	}
	log.Infof("Successfully wrote client bundle to %s", bundleDir)
	return nil
}
