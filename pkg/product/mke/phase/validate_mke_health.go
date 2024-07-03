package phase

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// ValidateMKEHealth validates MKE health locally from the MKE leader.
type ValidateMKEHealth struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *ValidateMKEHealth) Title() string {
	return "Validating MKE Health"
}

// Run validates the health of MKE is sane before continuing with other
// launchpad phases, should be used when installing products that depend
// on MKE, such as MSR.
func (p *ValidateMKEHealth) Run() error {
	// Issue a health check to the MKE local address managers until we receive an 'ok' status
	managers := p.Config.Spec.Managers()

	if err := p.Config.Spec.CheckMKEHealthLocal(managers); err != nil {
		return fmt.Errorf("%w: failed to validate MKE health: %w", errValidationFailed, err)
	}

	retries := p.Config.Spec.MKE.NodesHealthRetry
	if retries > 0 {
		h := p.Config.Spec.Managers()[0]

		tlsConfig, err := mke.GetTLSConfigFrom(h, p.Config.Spec.MKE.ImageRepo, p.Config.Spec.MKE.Version)
		if err != nil {
			return fmt.Errorf("error getting TLS config: %w", err)
		}

		url, err := p.Config.Spec.MKEURL()
		if err != nil {
			return fmt.Errorf("%w: get mke url: %w", errValidationFailed, err)
		}

		user := p.Config.Spec.MKE.AdminUsername
		if user == "" {
			return fmt.Errorf("%w: config Spec.MKE.AdminUsername not set", errValidationFailed)
		}
		pass := p.Config.Spec.MKE.AdminPassword
		if pass == "" {
			return fmt.Errorf("%w: config Spec.MKE.AdminPassword not set", errValidationFailed)
		}

		delay, _ := time.ParseDuration("10s")
		// Retry for total of 150 seconds
		err = retry.Do(
			func() error {
				log.Infof("%s: waiting for MKE nodes to become healthy", h)
				if err := checkMKENodesReady(url, tlsConfig, user, pass); err != nil {
					return fmt.Errorf("mke not ready: %w", err)
				}
				return nil
			},
			retry.Attempts(retries), retry.Delay(delay),
		)
		if err != nil {
			return fmt.Errorf("%w: failed to validate MKE health: %w", errValidationFailed, err)
		}
	}
	return nil
}

var (
	errRequestFailed = errors.New("request failed")
	errNodeNotReady  = errors.New("node not ready")
)

// checkMKENodesReady verifies the MKE nodes are in 'ready' state.
func checkMKENodesReady(mkeURL *url.URL, tlsConfig *tls.Config, username, password string) error {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Login and get a token for the user
	token, err := mke.GetToken(client, mkeURL, username, password)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	mkeURL.Path = "/nodes"

	// Perform the request
	req, err := http.NewRequest(http.MethodGet, mkeURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", mkeURL.String(), err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		log.Debugf("Failed to get response from %s: %v", mkeURL.String(), err)
		return fmt.Errorf("failed to get response from %s: %w", mkeURL.String(), err)
	}
	if err != nil {
		return fmt.Errorf("failed to poll /nodes endpoint. (%d): %w", resp.StatusCode, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: failed to poll /nodes endpoint. (http %d)", errRequestFailed, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var nodes []api.Node
	if err := json.Unmarshal(body, &nodes); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	for _, node := range nodes {
		if !node.IsReady() {
			log.Debugf("node %+v is not in ready state. State: '%+s'", node, node.Status.State)
			return fmt.Errorf("%w: node %+v is in state '%+s'", errNodeNotReady, node, node.Status.State)
		}
	}

	return nil
}
