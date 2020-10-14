package ucp

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	log "github.com/sirupsen/logrus"
)

// AuthToken represents a session token
type AuthToken struct {
	ID        string `json:"token_id,omitempty"`
	Token     string `json:"auth_token,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// Credentials represents a username/password pair for UCP login
type Credentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// CollectUcpFacts gathers the current status of installed UCP setup
// Currently we only need to know the existing version and whether UCP is installed or not.
// In future we probably need more.
func CollectUcpFacts(swarmLeader *api.Host, ucpMeta *api.UcpMetadata) error {
	output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`inspect --format '{{.Config.Image}}' ucp-proxy`))
	if err != nil {
		if strings.Contains(output, "No such object") {
			ucpMeta.Installed = false
			ucpMeta.InstalledVersion = ""
			return nil
		}
		return err
	}
	vparts := strings.Split(output, ":")
	if len(vparts) != 2 {
		return fmt.Errorf("malformed version output: %s", output)
	}

	ucpMeta.Installed = true
	ucpMeta.InstalledVersion = vparts[1]

	// Find out calico data plane by inspecting the calico container's env variables
	cmd := swarmLeader.Configurer.DockerCommandf(`ps --filter label=name="Calico node" --format {{.ID}}`)
	calicoContainer, err := swarmLeader.ExecWithOutput(cmd)

	if calicoContainer != "" && err != nil {
		log.Debugf("%s: calico container found: %s", swarmLeader, calicoContainer)
		cmd := swarmLeader.Configurer.DockerCommandf(`inspect %s --format {{.Config.Env}}`, calicoContainer)
		err := swarmLeader.Exec(fmt.Sprintf("%s | tr ' ' '\n' | grep FELIX_VXLAN= | grep -q true", cmd))
		if err != nil {
			ucpMeta.VXLAN = true
			log.Debugf("%s: has calico VXLAN enabled", swarmLeader)
		}
	}

	return nil
}

// GetClientBundle fetches the client bundle from UCP
// It returns a *zip.Reader with the contents of the bundle
// or an error if such occurred.
func GetClientBundle(ucpURL *url.URL, tlsConfig *tls.Config, username, password string) (*zip.Reader, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Login and get a token for the user
	token, err := GetUCPToken(client, ucpURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("Failed to get token for (%s:%s) : %s", username, password, err)
	}

	ucpURL.Path = "/api/clientbundle"

	// Now download the bundle
	req, err := http.NewRequest(http.MethodGet, ucpURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.Debugf("Failed to get bundle: %v", err)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if err == nil {
			return nil, fmt.Errorf("Failed to get client bundle (%d): %s", resp.StatusCode, string(body))
		}
		return nil, err
	}
	return zip.NewReader(bytes.NewReader(body), int64(len(body)))
}

// GetUCPToken gets a UCP Authtoken from the given ucpURL
func GetUCPToken(client *http.Client, ucpURL *url.URL, username, password string) (string, error) {
	ucpURL.Path = "/auth/login"
	creds := Credentials{
		Username: username,
		Password: password,
	}

	reqJSON, err := json.Marshal(creds)
	if err != nil {
		return "", err
	}
	resp, err := client.Post(ucpURL.String(), "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		log.Debugf("Failed to POST %s: %v", ucpURL.String(), err)
		return "", err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode == 200 {
		var authToken AuthToken
		if err := json.Unmarshal(body, &authToken); err != nil {
			return "", err
		}
		return authToken.Token, nil
	}
	return "", fmt.Errorf("Unexpected error logging in to UCP: %s", string(body))
}

// GetTLSConfigFrom retrieves the valid tlsConfig from the given UCP manager
func GetTLSConfigFrom(manager *api.Host, imageRepo, ucpVersion string) (*tls.Config, error) {
	runFlags := []string{"--rm", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if manager.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	output, err := manager.ExecWithOutput(fmt.Sprintf(`sudo docker run %s %s/ucp:%s dump-certs --ca`, strings.Join(runFlags, " "), imageRepo, ucpVersion))
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
