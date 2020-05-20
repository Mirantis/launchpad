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

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/ucp"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/Mirantis/mcc/pkg/config"
)

// Download downloads a UCP client bundle
func Download(ctx *cli.Context) error {
	cfgData, err := config.ResolveClusterFile(ctx)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = clusterConfig.Validate(); err != nil {
		return err
	}

	m := clusterConfig.Managers()[0]
	if err := m.Connect(); err != nil {
		return fmt.Errorf("error while connecting to manager node: %w", err)
	}

	tlsConfig, err := getTLSConfigFrom(m, clusterConfig.Ucp.Version)
	if err != nil {
		return fmt.Errorf("error getting TLS config: %w", err)
	}

	url, err := resolveURL(clusterConfig.Hosts[0].Address)
	if err != nil {
		return fmt.Errorf("error while parsing URL: %w", err)
	}
	username, password := ctx.String("username"), ctx.String("password")
	bundle, err := ucp.GetClientBundle(url, tlsConfig, username, password)
	if err != nil {
		return fmt.Errorf("failed to download admin bundle: %s", err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error while getting user home dir: %w", err)
	}
	err = writeBundle(homeDir, constant.StateBaseDir, "bundle", bundle)
	if err != nil {
		return fmt.Errorf("failed to write admin bundle: %s", err)
	}

	return nil
}

func resolveURL(serverURL string) (*url.URL, error) {
	if serverURL == "127.0.0.1" || serverURL == "localhost" {
		serverURL = "https://" + serverURL
	}
	url, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func getTLSConfigFrom(manager *config.Host, ucpVersion string) (*tls.Config, error) {
	cmd := fmt.Sprintf(`docker run --rm -v /var/run/docker.sock:/var/run/docker.sock docker/ucp:%s dump-certs --ca`, ucpVersion)
	output, err := manager.ExecWithOutput(cmd)
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

func writeBundle(bundleRoot, systemName, username string, bundle *zip.Reader) error {
	bundleDir := filepath.Join(bundleRoot, systemName, username)
	_, err := os.Stat(bundleDir)
	if err == nil {
		log.Debugf("Detected existing bundle at %s, removing", bundleDir)
		err := os.RemoveAll(bundleDir)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("unexpected error looking for bundle dir: %s", err)
	}
	err = os.MkdirAll(bundleDir, 0700)
	if err != nil {
		return fmt.Errorf("Failed to create bundle dir: %s", err)
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
	return nil
}
