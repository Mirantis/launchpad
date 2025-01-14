package integration_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

var AWS = map[string]interface{}{
	"region": "us-east-1",
}

var MKE_CONNECT = map[string]interface{}{
	"username": "admin",
	"password": "",
	"insecure": false,
}

var LAUNCHPAD = map[string]interface{}{
	"drain":       false,
	"mcr_version": "23.0.15",
	"mke_version": "3.7.15",
	"msr_version": "",
	"mke_connect": MKE_CONNECT,
}

// TestMain function to control the test execution
func TestMain(m *testing.M) {
	t := &testing.T{}
	log.Println("TestMKEClientConfig")

	name := fmt.Sprintf("integration-%s", test.GenerateRandomAlphaNumericString(5))
	MKE_CONNECT["password"] = test.GenerateRandomAlphaNumericString(12)

	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../examples/terraform/aws-simple",
		VarFiles:     []string{"smoke-small.tfvars"},
		Vars: map[string]interface{}{
			"name":            name,
			"aws":             AWS,
			"launchpad":       LAUNCHPAD,
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

// Tests below will run in the scope of the above TestMain cluster

func TestMKEClientConfig(t *testing.T) {
	sp := test.GetInstance()
	product := sp.Product

	err := product.ClientConfig()
	assert.NoError(t, err)

	home, err := homedir.Dir()
	assert.NoError(t, err)

	t.Logf("Terraform Options: %v", sp.TerraformOptions)
	mkeConnectOut := terraform.OutputJson(t, sp.TerraformOptions, "mke_connect")

	product.ClientConfig()
	var m struct {
		mke.Credentials
		Host     string
		Insecure bool
	}

	err = json.Unmarshal([]byte(mkeConnectOut), &m)
	assert.NoError(t, err)

	bundlePath := path.Join(home, constant.StateBaseDir, "cluster", product.ClusterName(), "bundle", m.Username)
	_, err = os.Stat(bundlePath)
	assert.NoError(t, err)

	err = os.RemoveAll(bundlePath)
	assert.NoError(t, err)
}
