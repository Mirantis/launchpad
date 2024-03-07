package smoke_test

import (
	"fmt"
	"log"
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
	"insecure": false,
}

var LAUNCHPAD = map[string]interface{}{
	"drain":       false,
	"mcr_version": "23.0.8",
	"mke_version": "3.7.3",
	"msr_version": "",
	"mke_connect": MKE_CONNECT,
}

// configure the network stack
var NETWORK = map[string]interface{}{
	"cidr":                 "172.31.0.0/16",
	"public_subnet_count":  3,
	"private_subnet_count": 0, // if 0 then no private nodegroups allowed
}

// TestSmallCluster deploys a small test cluster
func TestSmallCluster(t *testing.T) {
	log.Println("TestSmallCluster")
	nodegroups := map[string]interface{}{
		"MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
		"WrkUbuntu22":  test.Platforms["Ubuntu22"].GetWorker(),
		"WrkWindows19": test.Platforms["Windows2019"].GetWorker(),
	}

	uTestId, err := test.GenerateRandomString(5)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("smoke-%s", uTestId)

	rndPassword, err := test.GenerateRandomString(12)
	if err != nil {
		t.Fatal(err)
	}

	MKE_CONNECT["password"] = rndPassword

	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../examples/tf-aws/launchpad",
		Vars: map[string]interface{}{
			"name":             name,
			"aws":              AWS,
			"launchpad":        LAUNCHPAD,
			"network":          NETWORK,
			"ssh_pk_location":  tempSSHKeyPathDir,
			"nodegroups":       nodegroups,
			"windows_password": rndPassword,
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

		"WrkUbuntu22":  test.Platforms["Ubuntu22"].GetWorker(),
		"WrkRocky9":    test.Platforms["Rocky9"].GetWorker(),
		"WrkSles15":    test.Platforms["Sles15"].GetWorker(),
		"WrkCentos7":   test.Platforms["Centos7"].GetWorker(),
		"WrkRhel9":     test.Platforms["Rhel9"].GetWorker(),
		"WrkOracle9":   test.Platforms["Oracle9"].GetWorker(),
		"WrkWindows22": test.Platforms["Windows2022"].GetWorker(),
	}

	uTestId, err := test.GenerateRandomString(5)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("smoke-%s", uTestId)

	rndPassword, err := test.GenerateRandomString(12)
	if err != nil {
		t.Fatal(err)
	}

	MKE_CONNECT["password"] = rndPassword

	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../examples/tf-aws/launchpad",
		Vars: map[string]interface{}{
			"name":             name,
			"aws":              AWS,
			"launchpad":        LAUNCHPAD,
			"network":          NETWORK,
			"ssh_pk_location":  tempSSHKeyPathDir,
			"nodegroups":       nodegroups,
			"windows_password": rndPassword,
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
