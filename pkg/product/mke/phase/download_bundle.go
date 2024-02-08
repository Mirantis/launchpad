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

var errInvalidConfig = fmt.Errorf("invalid config")

// Run collect all the facts from hosts in parallel.
func (p *DownloadBundle) Run() error {
	m := p.Config.Spec.Managers()[0]

	tlsConfig, err := mke.GetTLSConfigFrom(m, p.Config.Spec.MKE.ImageRepo, p.Config.Spec.MKE.Version)
	if err != nil {
		return fmt.Errorf("error getting TLS config: %w", err)
	}

	url, err := p.Config.Spec.MKEURL()
	if err != nil {
		return fmt.Errorf("get mke url: %w", err)
	}

	user := p.Config.Spec.MKE.AdminUsername
	if user == "" {
		return fmt.Errorf("%w: config Spec.MKE.AdminUsername not set", errInvalidConfig)
	}
	pass := p.Config.Spec.MKE.AdminPassword
	if pass == "" {
		return fmt.Errorf("%w: config Spec.MKE.AdminPassword not set", errInvalidConfig)
	}

	bundle, err := mke.GetClientBundle(url, tlsConfig, user, pass)
	if err != nil {
		return fmt.Errorf("failed to download admin bundle: %w", err)
	}

	bundleDir, err := p.getBundleDir(p.Config.Metadata.Name, user)
	if err != nil {
		return fmt.Errorf("failed to get bundle directory: %w", err)
	}
	err = p.writeBundle(bundleDir, bundle)
	if err != nil {
		return fmt.Errorf("failed to write admin bundle: %w", err)
	}

	return nil
}

func (p *DownloadBundle) getBundleDir(clusterName, username string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return path.Join(home, constant.StateBaseDir, "cluster", clusterName, "bundle", username), nil
}

var errInvalidBundle = fmt.Errorf("invalid bundle")

func safePath(base, rel string) (string, error) {
	abs, err := filepath.Abs(filepath.Join(base, rel))
	if err != nil {
		return "", fmt.Errorf("%w: error while getting absolute path: %w", errInvalidBundle, err)
	}
	if !strings.HasPrefix(abs, base) {
		return "", fmt.Errorf("%w: zip slip detected", errInvalidBundle)
	}
	return abs, nil
}

func (p *DownloadBundle) writeBundle(bundleDir string, bundle *zip.Reader) error {
	if err := util.EnsureDir(bundleDir); err != nil {
		return fmt.Errorf("error while creating directory: %w", err)
	}
	log.Debugf("Writing out bundle to %s", bundleDir)
	for _, zipFile := range bundle.File {
		src, err := zipFile.Open()
		if err != nil {
			return fmt.Errorf("error while opening file %s: %w", zipFile.Name, err)
		}
		defer src.Close()
		var data []byte
		data, err = io.ReadAll(src)
		if err != nil {
			return fmt.Errorf("error while reading file %s: %w", zipFile.Name, err)
		}
		mode := int64(0o644)
		if strings.Contains(zipFile.Name, "key.pem") {
			mode = 0o600
		}

		// mke bundle will contain folders as well as files, if folder exists fd will not be empty
		if dir := filepath.Dir(zipFile.Name); dir != "" && dir != "." {
			if err := os.MkdirAll(filepath.Join(bundleDir, dir), 0o700); err != nil {
				return fmt.Errorf("error while creating directory: %w", err)
			}
		}

		outFile, err := safePath(bundleDir, zipFile.Name)
		if err != nil {
			return err
		}

		err = os.WriteFile(outFile, data, os.FileMode(mode))
		if err != nil {
			return fmt.Errorf("error while writing file %s: %w", zipFile.Name, err)
		}
	}
	log.Infof("Successfully wrote client bundle to %s", bundleDir)
	return nil
}
