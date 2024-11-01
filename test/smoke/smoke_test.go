package smoke_test

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

var AWS = map[string]interface{}{
	"region": "us-east-1",
}

var MKE_CONNECT = map[string]interface{}{
	"username": "admin",
	"password": "",
	"insecure": true,
}

var LAUNCHPAD = map[string]interface{}{
	"drain":       false,
	"mcr_version": "23.0.7",
	"mke_version": "3.7.3",
	"msr_version": "2.9.16",
	"mke_connect": MKE_CONNECT,
}

// configure the network stack
var NETWORK = map[string]interface{}{
	"cidr": "172.31.0.0/16",
}
var SUBNETS = map[string]interface{}{
	"main": map[string]interface{}{
		"cidr":       "172.31.0.0/17",
		"private":    false,
		"nodegroups": []string{"MngrUbuntu22", "MngrRocky9", "MngrSles15", "MngrCentos7", "MngrRhel9", "WrkUbuntu22", "WrkRocky9", "WrkSles15", "WrkCentos7", "WrkRhel9"},
	},
}

// TestSmallCluster deploys a small test cluster
func TestSmallCluster(t *testing.T) {
	log.Println("TestSmallCluster")

	nodegroups := map[string]interface{}{
		"MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
		"WrkUbuntu22":  test.Platforms["Ubuntu22"].GetWorker(),
	}

	uTestId := test.GenerateRandomAlphaNumericString(5)

	name := fmt.Sprintf("smoke-%s", uTestId)

	rndPassword := test.GenerateRandomAlphaNumericString(12)

	MKE_CONNECT["password"] = rndPassword

	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../examples/tf-aws/launchpad",
		Vars: map[string]interface{}{
			"name":            name,
			"aws":             AWS,
			"launchpad":       LAUNCHPAD,
			"network":         NETWORK,
			"subnets":         SUBNETS,
			"ssh_pk_location": tempSSHKeyPathDir,
			"nodegroups":      nodegroups,
		},
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &options)
	// Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	if _, err := terraform.InitAndApplyE(t, terraformOptions); err != nil {
		t.Fatal(err)
	}

	// Destroy the Terraform resources at the end of the test
	defer terraform.Destroy(t, terraformOptions)

	mkeClusterConfig := terraform.Output(t, terraformOptions, "launchpad_yaml")

	product, err := config.ProductFromYAML([]byte(mkeClusterConfig))
	assert.NoError(t, err)

	// Do Launchpad Apply as pre-requisite to the tests
	err = product.Apply(true, true, 3, true)
	assert.NoError(t, err)

	err = product.Reset()
	assert.NoError(t, err)
}

// TestSupportedMatrixCluster deploys a cluster with all supported platforms
func TestSupportedMatrixCluster(t *testing.T) {
	log.Println("TestSupportedMatrixCluster")

	nodegroups := map[string]interface{}{
		"MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
		"MngrRocky9":   test.Platforms["Rocky9"].GetManager(),
		"MngrSles15":   test.Platforms["Sles15"].GetManager(),
		"MngrCentos7":  test.Platforms["Centos7"].GetManager(),
		"MngrRhel9":    test.Platforms["Rhel9"].GetManager(),

		"WrkUbuntu22": test.Platforms["Ubuntu22"].GetWorker(),
		"WrkRocky9":   test.Platforms["Rocky9"].GetWorker(),
		"WrkSles15":   test.Platforms["Sles15"].GetWorker(),
		"WrkCentos7":  test.Platforms["Centos7"].GetWorker(),
		"WrkRhel9":    test.Platforms["Rhel9"].GetWorker(),
	}

	uTestId := test.GenerateRandomAlphaNumericString(5)

	name := fmt.Sprintf("smoke-%s", uTestId)

	rndPassword := test.GenerateRandomAlphaNumericString(12)

	MKE_CONNECT["password"] = rndPassword

	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../examples/tf-aws/launchpad",
		Vars: map[string]interface{}{
			"name":            name,
			"aws":             AWS,
			"launchpad":       LAUNCHPAD,
			"network":         NETWORK,
			"subnets":         SUBNETS,
			"ssh_pk_location": tempSSHKeyPathDir,
			"nodegroups":      nodegroups,
		},
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &options)
	// Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	if _, err := terraform.InitAndApplyE(t, terraformOptions); err != nil {
		t.Fatal(err)
	}

	// Destroy the Terraform resources at the end of the test
	defer terraform.Destroy(t, terraformOptions)

	mkeClusterConfig := terraform.Output(t, terraformOptions, "launchpad_yaml")

	product, err := config.ProductFromYAML([]byte(mkeClusterConfig))
	assert.NoError(t, err)

	// Do Launchpad Apply as pre-requisite to the tests
	err = product.Apply(true, true, 3, true)
	assert.NoError(t, err)

	// Replace the version values for MCR,MKE,MSR in the mkeClusterConfig
	mkeClusterConfig = strings.ReplaceAll(mkeClusterConfig, LAUNCHPAD["mcr_version"].(string), "23.0.9")
	mkeClusterConfig = strings.ReplaceAll(mkeClusterConfig, LAUNCHPAD["mke_version"].(string), "3.7.5")

	productUpgrade, err := config.ProductFromYAML([]byte(mkeClusterConfig))
	assert.NoError(t, err)

	err = productUpgrade.Apply(true, true, 3, true)
	assert.NoError(t, err)

	err = product.Reset()
	assert.NoError(t, err)
}
