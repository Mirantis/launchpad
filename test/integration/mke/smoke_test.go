package mke_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	test "github.com/Mirantis/mcc/test/integration"
)

// TestMain function to control the test execution
func TestMain(m *testing.M) {
	t := &testing.T{}
	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../../examples/tf-aws/launchpad",
		Vars: map[string]interface{}{
			"name": "test",
			"aws": map[string]interface{}{
				"region": "eu-central-1",
			},
			"ssh_pk_location": tempSSHKeyPathDir,
		},
	}
	sp := test.GetInstance()
	sp.Setup(t, options)

	// Do Launchpad Apply as pre-requisite to the tests
	err := sp.Product.Apply(true, true, 3, true)
	assert.NoError(t, err)

	// Run tests in order
	code := m.Run()

	// Teardown
	test.Destroy()

	// Exit with the status code of the test run
	os.Exit(code)
}

func TestMKEPing(t *testing.T) {
	sp := test.GetInstance()

	mkeConnectOut := terraform.OutputJson(t, &sp.TerraformOptions, "mke_connect")

	var m test.MKEConnect
	err := json.Unmarshal([]byte(mkeConnectOut), &m)
	assert.NoError(t, err)

	url := fmt.Sprintf("https://%s/_ping", m.Host)

	client := test.GetHTTPClient(&tls.Config{InsecureSkipVerify: true})

	// Make the request
	resp, err := test.UnAuthHttpRequest(url, http.MethodGet, nil, client)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestMKENodesReady(t *testing.T) {
	sp := test.GetInstance()

	mkeConnectOut := terraform.OutputJson(t, &sp.TerraformOptions, "mke_connect")

	var m test.MKEConnect
	err := json.Unmarshal([]byte(mkeConnectOut), &m)
	assert.NoError(t, err)

	client := test.GetHTTPClient(&tls.Config{InsecureSkipVerify: true})

	// Get token
	mkeURL, err := url.Parse(fmt.Sprintf("https://%s", m.Host))
	assert.NoError(t, err)
	token, err := mke.GetToken(client, mkeURL, m.Username, m.Password)
	assert.NoError(t, err)

	endpointURL := fmt.Sprintf("https://%s/nodes", m.Host)

	nodesReady := false
	// Wait 100 seconds for nodes to be ready
	for i := 0; i < 10; i++ {
		// Make the request
		resp, err := test.AuthHttpRequest(endpointURL, token, http.MethodGet, nil, client)
		if err != nil {
			break
		}

		nodes := []api.Node{}
		err = json.NewDecoder(resp.Body).Decode(&nodes)
		assert.NoError(t, err)

		nodesReady = true
		for _, node := range nodes {
			if !node.IsReady() {
				nodesReady = false
				fmt.Printf("Node %s is not ready\n", node.Description.Hostname)
				break
			}
		}
		defer resp.Body.Close()
		if nodesReady {
			break
		}
		fmt.Printf("Nodes are not ready, retrying in 10 seconds\n")
		time.Sleep(10 * time.Second)
	}

	assert.True(t, nodesReady)
}

func TestMKEClientConfig(t *testing.T) {
	sp := test.GetInstance()

	err := sp.Product.ClientConfig()
	assert.NoError(t, err)

	home, err := homedir.Dir()
	assert.NoError(t, err)
	mkeConnectOut := terraform.OutputJson(t, &sp.TerraformOptions, "mke_connect")

	var m test.MKEConnect
	err = json.Unmarshal([]byte(mkeConnectOut), &m)
	assert.NoError(t, err)

	bundlePath := path.Join(home, constant.StateBaseDir, "cluster", sp.Product.ClusterName(), "bundle", m.Username)
	_, err = os.Stat(bundlePath)
	assert.NoError(t, err)
	err = os.RemoveAll(bundlePath)
	assert.NoError(t, err)
}

func TestLaunchpadReset(t *testing.T) {
	sp := test.GetInstance()

	err := sp.Product.Reset()
	assert.NoError(t, err)
}
