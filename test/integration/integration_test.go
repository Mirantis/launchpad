package integration_test

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"testing"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/test"
	"github.com/mitchellh/go-homedir"
	"github.com/stretchr/testify/assert"
)

var AWS = map[string]any{
	"region": "us-east-1",
}

var MKE_CONNECT = map[string]any{
	"username": "admin",
	"password": "",
	"insecure": false,
}

var LAUNCHPAD = map[string]any{
	"drain":       false,
	"mcr_channel": "stable-25.0",
	"mke_version": "3.8.8",
	"msr_version": "2.9.28",
	"mke_connect": MKE_CONNECT,
}

// configure the network stack
var NETWORK = map[string]any{
	"cidr": "172.31.0.0/16",
}

// One subnet per AZ, each nodegroup in exactly one of them. The provision
// module derives AZs as slice(available_azs, 0, len(subnets)), so the subnet
// count is the AZ count; a single subnet put every nodegroup in one AZ and
// made the stack fail when that AZ ran out of the instance type. The CIDRs
// subdivide the /17 the single subnet used, leaving the VPC's upper half free.
var SUBNETS = map[string]any{
	"main0": map[string]any{
		"cidr":       "172.31.0.0/19",
		"private":    false,
		"nodegroups": []string{"MngrUbuntu22", "WrkUbuntu22"},
	},
	"main1": map[string]any{
		"cidr":       "172.31.32.0/19",
		"private":    false,
		"nodegroups": []string{"WrkRhel9"},
	},
	"main2": map[string]any{
		"cidr":       "172.31.64.0/19",
		"private":    false,
		"nodegroups": []string{"WrkSles15"},
	},
}

// TestMain function to control the test execution
func TestMain(m *testing.M) {
	t := &testing.T{}
	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	log.Println("TestMKEClientConfig")
	nodegroups := map[string]any{
		"MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
		"WrkRhel9":     test.Platforms["Ubuntu22"].GetWorker(),
		"WrkUbuntu22":  test.Platforms["Rhel9"].GetWorker(),
		"WrkSles15":    test.Platforms["Sles15"].GetWorker(),
	}

	uTestId := test.GenerateRandomAlphaNumericString(5)

	name := fmt.Sprintf("smoke-%s", uTestId)

	mkePassword := test.GenerateRandomAlphaNumericString(12)
	MKE_CONNECT["password"] = mkePassword

	options := test.TerraformOptions(
		t,
		// The path to where the Terraform tf chart is located
		"../../examples/terraform/aws-simple",
		map[string]any{
			"name":            name,
			"aws":             AWS,
			"admin_cidrs":     test.AdminCIDRs(t),
			"network":         NETWORK,
			"subnets":         SUBNETS,
			"ssh_pk_location": tempSSHKeyPathDir,
			"nodegroups":      nodegroups,
		},
		// LAUNCHPAD nests the MKE admin password, so it travels by TF_VAR_*
		// rather than as a "-var" argument. See test.TerraformOptions.
		map[string]any{
			"launchpad": LAUNCHPAD,
		},
	)

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

func TestMKEClientConfig(t *testing.T) {
	sp := test.GetInstance()
	product := sp.Product

	err := product.ClientConfig()
	assert.NoError(t, err)

	home, err := homedir.Dir()
	assert.NoError(t, err)

	mkeConnectOut := test.OutputSecretJSON(t, sp.TerraformOptions, "mke_connect")

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
