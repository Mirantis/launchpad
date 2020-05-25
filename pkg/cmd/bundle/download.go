package bundle

import (
	"archive/zip"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/state"
	"github.com/Mirantis/mcc/pkg/ucp"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Download downloads a UCP client bundle
func Download(clusterFile string, username string, password string) error {
	cfgData, err := config.ResolveClusterFile(clusterFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = config.Validate(&clusterConfig); err != nil {
		return err
	}

	m := clusterConfig.Spec.Managers()[0]
	if err := m.Connect(); err != nil {
		return fmt.Errorf("error while connecting to manager node: %w", err)
	}

	tlsConfig, err := getTLSConfigFrom(m, clusterConfig.Spec.Ucp.ImageRepo, clusterConfig.Spec.Ucp.Version)
	if err != nil {
		return fmt.Errorf("error getting TLS config: %w", err)
	}

	url, err := resolveURL(clusterConfig.Spec.Hosts[0].Address)
	if err != nil {
		return fmt.Errorf("error while parsing URL: %w", err)
	}
	bundle, err := ucp.GetClientBundle(url, tlsConfig, username, password)
	if err != nil {
		return fmt.Errorf("failed to download admin bundle: %s", err)
	}
	// Need to initilize state directly, clusterConfig does not have it initilized outside of the phases which we do not use here.
	state := &state.State{
		Name: clusterConfig.Metadata.Name,
	}
	stateDir, err := state.GetDir()
	if err != nil {
		return fmt.Errorf("failed to get state directory for cluster %s: %w", clusterConfig.Metadata.Name, err)
	}

	err = writeBundle(filepath.Join(stateDir, "bundle", username), bundle)
	if err != nil {
		return fmt.Errorf("failed to write admin bundle: %s", err)
	}

	return nil
}

func resolveURL(serverURL string) (*url.URL, error) {
	url, err := url.Parse(fmt.Sprintf("https://%s", serverURL))
	if err != nil {
		return nil, err
	}
	return url, nil
}

func getTLSConfigFrom(manager *api.Host, imageRepo, ucpVersion string) (*tls.Config, error) {
	output, err := manager.ExecWithOutput(fmt.Sprintf(`docker run --rm -v /var/run/docker.sock:/var/run/docker.sock %s/ucp:%s dump-certs --ca`, imageRepo, ucpVersion))
	if err != nil {
		return nil, fmt.Errorf("error while exec-ing into the container: %w", err)
	}
	i := strings.Index(output, "-----BEGIN CERTIFICATE-----")
	if i < 0 {
		return nil, fmt.Errorf("malformed certificate")
	}

	cert := []byte(output[i:])
	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, fmt.Errorf("error while appending certs to PEM")
	}
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}

func writeBundle(bundleDir string, bundle *zip.Reader) error {
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
	log.Infof("Succesfully wrote client bundle to %s", bundleDir)
	return nil
}
