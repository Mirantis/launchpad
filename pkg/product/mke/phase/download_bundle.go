package phase

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
)

// DownloadBundle phase downloads the client bundle to local storage.
type DownloadBundle struct {
	phase.BasicPhase
}

// Title for the phase.
func (p *DownloadBundle) Title() string {
	return "Download Client Bundle"
}

// Run collect all the facts from hosts in parallel.
func (p *DownloadBundle) Run() error {
	m := p.Config.Spec.Managers()[0]

	tlsConfig, err := mke.GetTLSConfigFrom(m, p.Config.Spec.MKE.ImageRepo, p.Config.Spec.MKE.Version)
	if err != nil {
		return fmt.Errorf("error getting TLS config: %w", err)
	}

	url, err := p.Config.Spec.MKEURL()
	if err != nil {
		return err
	}

	user := p.Config.Spec.MKE.AdminUsername
	if user == "" {
		return fmt.Errorf("config Spec.MKE.AdminUsername not set")
	}
	pass := p.Config.Spec.MKE.AdminPassword
	if pass == "" {
		return fmt.Errorf("config Spec.MKE.AdminPassword not set")
	}

	bundle, err := mke.GetClientBundle(url, tlsConfig, user, pass)
	if err != nil {
		return fmt.Errorf("failed to download admin bundle: %s", err)
	}

	bundleDir, err := p.getBundleDir(p.Config.Metadata.Name, user)
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
		data, err = io.ReadAll(src)
		if err != nil {
			return err
		}
		mode := int64(0o644)
		if strings.Contains(zf.Name, "key.pem") {
			mode = 0o600
		}

		// mke bundle will contain folders as well as files, if folder exists fd will not be empty
		dir, _ := filepath.Split(zf.Name)
		if dir != "" {
			if err := os.MkdirAll(filepath.Join(bundleDir, dir), 0o700); err != nil {
				return err
			}
		}

		err = os.WriteFile(filepath.Join(bundleDir, zf.Name), data, os.FileMode(mode))
		if err != nil {
			return err
		}
	}
	log.Infof("Successfully wrote client bundle to %s", bundleDir)
	return nil
}
