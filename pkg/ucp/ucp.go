package ucp

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
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
func CollectUcpFacts(swarmLeader *config.Host) (*config.UcpMetadata, error) {
	output, err := swarmLeader.ExecWithOutput(swarmLeader.Configurer.DockerCommandf(`inspect --format '{{ index .Config.Labels "com.docker.ucp.version"}}' ucp-proxy`))
	if err != nil {
		// We need to check the output to check if the container does not exist
		if strings.Contains(output, "No such object") {
			return &config.UcpMetadata{Installed: false}, nil
		}
		return nil, err
	}
	ucpMeta := &config.UcpMetadata{
		Installed:        true,
		InstalledVersion: output,
	}
	return ucpMeta, nil
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
	token, err := getUCPToken(client, ucpURL, username, password)
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

func getUCPToken(client *http.Client, ucpURL *url.URL, username, password string) (string, error) {
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
