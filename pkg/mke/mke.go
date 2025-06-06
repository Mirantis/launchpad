package mke

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mirantis/launchpad/pkg/constant"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/Mirantis/launchpad/pkg/product/mke/api"
	"github.com/hashicorp/go-version"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// AuthToken represents a session token.
type AuthToken struct {
	ID        string `json:"token_id,omitempty"`
	Token     string `json:"auth_token,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// Credentials represents a username/password pair for mke login.
type Credentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

var errInvalidVersion = errors.New("invalid version")

// CollectFacts gathers the current status of installed mke setup.
func CollectFacts(swarmLeader *api.Host, mkeMeta *api.MKEMetadata) error {
	output, err := swarmLeader.ExecOutput(swarmLeader.Configurer.DockerCommandf(`inspect --format '{{.Config.Image}}' ucp-proxy`))
	if err != nil {
		mkeMeta.Installed = false
		mkeMeta.InstalledVersion = ""
		return nil
	}

	vparts := strings.Split(output, ":")
	if len(vparts) != 2 {
		return fmt.Errorf("%w: malformed version output: %s", errInvalidVersion, output)
	}
	repo := vparts[0][:strings.LastIndexByte(vparts[0], '/')]

	mkeMeta.Installed = true
	mkeMeta.InstalledVersion = vparts[1]
	mkeMeta.InstalledBootstrapImage = fmt.Sprintf("%s:/ucp:%s", repo, vparts[1])

	// Find out calico data plane by inspecting the calico container's env variables
	cmd := swarmLeader.Configurer.DockerCommandf(`ps --filter label=name="Calico node" --format {{.ID}}`)
	calicoContainer, err := swarmLeader.ExecOutput(cmd)

	if calicoContainer != "" && err != nil {
		log.Debugf("%s: calico container found: %s", swarmLeader, calicoContainer)
		cmd := swarmLeader.Configurer.DockerCommandf(`inspect %s --format {{.Config.Env}}`, calicoContainer)
		err := swarmLeader.Exec(fmt.Sprintf("%s | tr ' ' '\n' | grep FELIX_VXLAN= | grep -q true", cmd))
		if err != nil {
			mkeMeta.VXLAN = true
			log.Debugf("%s: has calico VXLAN enabled", swarmLeader)
		}
	}

	return nil
}

// GetClientBundle fetches the client bundle from mke
// It returns a *zip.Reader with the contents of the bundle
// or an error if such occurred.
func GetClientBundle(mkeURL *url.URL, tlsConfig *tls.Config, username, password string) (*zip.Reader, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Login and get a token for the user
	token, err := GetToken(client, mkeURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to get token for (%s:%s): %w", username, password, err)
	}

	mkeURL.Path = "/api/clientbundle"

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Now download the bundle
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mkeURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.Debugf("Failed to get bundle: %v", err)
		return nil, fmt.Errorf("failed to request client bundle: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to read client bundle (%d): %s: %w", resp.StatusCode, string(body), err)
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create a reader for client bundle: %w", err)
	}
	return reader, nil
}

var errGetToken = errors.New("failed to get token")

// GetToken gets a mke Authtoken from the given mkeURL.
func GetToken(client *http.Client, mkeURL *url.URL, username, password string) (string, error) {
	mkeURL.Path = "/auth/login"
	creds := Credentials{
		Username: username,
		Password: password,
	}

	reqJSON, err := json.Marshal(creds)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credentials: %w", err)
	}
	resp, err := client.Post(mkeURL.String(), "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		log.Debugf("Failed to POST %s: %v", mkeURL.String(), err)
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		var authToken AuthToken
		if err := json.Unmarshal(body, &authToken); err != nil {
			return "", fmt.Errorf("failed to unmarshal token response: %w", err)
		}
		return authToken.Token, nil
	}
	return "", fmt.Errorf("%w: unexpected error logging in to mke: %s", errGetToken, string(body))
}

var errGetTLSConfig = errors.New("failed to get TLS config")

// GetTLSConfigFrom retrieves the valid tlsConfig from the given mke manager.
func GetTLSConfigFrom(manager *api.Host, imageRepo, mkeVersion string) (*tls.Config, error) {
	runFlags := common.Flags{"--rm", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if manager.Configurer.SELinuxEnabled(manager) {
		runFlags.Add("--security-opt label=disable")
	}
	output, err := manager.ExecOutput(manager.Configurer.DockerCommandf(`run %s %s/ucp:%s dump-certs --ca`, runFlags.Join(), imageRepo, mkeVersion), exec.Redact(`[A-Za-z0-9+/=_\-]{64}`))
	if err != nil {
		return nil, fmt.Errorf("%w: error while exec-ing into the container: %w", errGetTLSConfig, err)
	}
	i := strings.Index(output, "-----BEGIN CERTIFICATE-----")
	if i < 0 {
		return nil, fmt.Errorf("%w: malformed certificate", errGetTLSConfig)
	}

	cert := []byte(output[i:])
	block, _ := pem.Decode(cert)
	if block == nil {
		return nil, fmt.Errorf("%w: no certificates found in output", errGetTLSConfig)
	}

	if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, fmt.Errorf("%w: invalid certificate: %#v", errGetTLSConfig, block)
	}
	if _, err = x509.ParseCertificate(block.Bytes); err != nil {
		return nil, fmt.Errorf("%w: failed to parse certificate: %w", errGetTLSConfig, err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, fmt.Errorf("%w: error while appending certs to PEM", errGetTLSConfig)
	}
	return &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}, nil
}

func tp2qp(s string) string {
	return strings.Replace(s, "-tp", "-qp", 1)
}

// VersionGreaterThan is a "corrected" version comparator that considers -tpX releases to be earlier than -rcX.
func VersionGreaterThan(a, b *version.Version) bool {
	ca, _ := version.NewVersion(tp2qp(a.String()))
	cb, _ := version.NewVersion(tp2qp(b.String()))
	return ca.GreaterThan(cb)
}

var (
	errNoManagersInConfig = errors.New("no managers found in config")
	errInvalidConfig      = errors.New("invalid config")
)

// DownloadBundle downloads the client bundle from MKE to local storage.
func DownloadBundle(config *api.ClusterConfig) error {
	if len(config.Spec.Managers()) == 0 {
		return errNoManagersInConfig
	}

	m := config.Spec.Managers()[0]

	tlsConfig, err := GetTLSConfigFrom(m, config.Spec.MKE.ImageRepo, config.Spec.MKE.Version)
	if err != nil {
		return fmt.Errorf("error getting TLS config: %w", err)
	}

	url, err := config.Spec.MKEURL()
	if err != nil {
		return fmt.Errorf("get mke url: %w", err)
	}

	log.Debugf("downloading admin bundle from MKE: %s", url)

	user := config.Spec.MKE.AdminUsername
	if user == "" {
		return fmt.Errorf("%w: config Spec.MKE.AdminUsername not set", errInvalidConfig)
	}

	pass := config.Spec.MKE.AdminPassword
	if pass == "" {
		return fmt.Errorf("%w: config Spec.MKE.AdminPassword not set", errInvalidConfig)
	}

	bundle, err := GetClientBundle(url, tlsConfig, user, pass)
	if err != nil {
		return fmt.Errorf("failed to download admin bundle: %w", err)
	}

	bundleDir, err := getBundleDir(config)
	if err != nil {
		return err
	}
	err = writeBundle(bundleDir, bundle)
	if err != nil {
		return fmt.Errorf("failed to write admin bundle: %w", err)
	}

	return nil
}

func getBundleDir(config *api.ClusterConfig) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return path.Join(home, constant.StateBaseDir, "cluster", config.Metadata.Name, "bundle", config.Spec.MKE.AdminUsername), nil
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

func writeBundle(bundleDir string, bundle *zip.Reader) error {
	if err := os.MkdirAll(bundleDir, os.ModePerm); err != nil {
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
		mode := uint32(0o644)
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
