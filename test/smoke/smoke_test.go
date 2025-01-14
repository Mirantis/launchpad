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
	"mcr_version": "23.0.15",
	"mke_version": "3.7.15",
	"msr_version": "2.9.16",
	"mke_connect": MKE_CONNECT,
}

// TestSmallCluster deploys a small test cluster
func TestSmallCluster(t *testing.T) {
	log.Println("TestSmallCluster")

	name := fmt.Sprintf("smoke-%s", test.GenerateRandomAlphaNumericString(5))
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

	name := fmt.Sprintf("smoke-%s", test.GenerateRandomAlphaNumericString(5))
	MKE_CONNECT["password"] = test.GenerateRandomAlphaNumericString(12)

	// Create a temporary directory to store Terraform files
	tempSSHKeyPathDir := t.TempDir()

	options := terraform.Options{
		// The path to where the Terraform tf chart is located
		TerraformDir: "../../examples/terraform/aws-simple",
		VarFiles:     []string{"smoke-full.tfvars"},
		Vars: map[string]interface{}{
			"name":            name,
			"aws":             AWS,
			"launchpad":       LAUNCHPAD,
			"ssh_pk_location": tempSSHKeyPathDir,
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
	mkeClusterConfig = strings.ReplaceAll(mkeClusterConfig, LAUNCHPAD["mcr_version"].(string), "23.0.16")
	mkeClusterConfig = strings.ReplaceAll(mkeClusterConfig, LAUNCHPAD["mke_version"].(string), "3.7.16")

	productUpgrade, err := config.ProductFromYAML([]byte(mkeClusterConfig))
	assert.NoError(t, err)

	err = productUpgrade.Apply(true, true, 3, true)
	assert.NoError(t, err)

	err = product.Reset()
	assert.NoError(t, err)
}
