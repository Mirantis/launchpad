package test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/product"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

type singletonProduct struct {
	Product          product.Product
	TerraformOptions *terraform.Options
	T                *testing.T
}

var (
	instance *singletonProduct
	once     sync.Once
)

// GetInstance returns the singleton instance.
//
//nolint:revive
func GetInstance() *singletonProduct {
	once.Do(func() {
		instance = &singletonProduct{}
	})
	return instance
}

// Destroy destroys the singleton instance.
func Destroy() {
	instance.Teardown()
	instance = nil
}

// Setup function to run before tests.
func (sp *singletonProduct) Setup(t *testing.T, options terraform.Options) {
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &options)
	// Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	if _, err := terraform.InitAndApplyE(t, terraformOptions); err != nil {
		t.Fatal(err)
	}

	mkeClusterConfig := terraform.Output(t, terraformOptions, "launchpad_yaml")

	product, err := config.ProductFromYAML([]byte(mkeClusterConfig))
	sp.Product = product
	sp.TerraformOptions = terraformOptions
	sp.T = t
	if err != nil {
		t.Cleanup(func() {
			sp.Teardown()
		})

		t.Fatalf("Error parsing launchpad yaml: %v\n", err)
	}
}

// Teardown function to run after tests.
func (sp *singletonProduct) Teardown() {
	fmt.Println("Teardown function executed")
	if sp == nil {
		return
	}
	// Destroy the Terraform resources
	terraform.Destroy(sp.T, sp.TerraformOptions)
}
