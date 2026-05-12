package smoke_test

// Upgrade smoke test: provision a cluster at a baseline version, upgrade it
// to a target version using a second Apply() call, verify the cluster is
// healthy, then tear down.
//
// Design notes
// ============
// Launchpad's Apply() is idempotent and version-aware: when the installed MCR
// channel or MKE version differs from the config, UpgradeMCR / UpgradeMKE
// phases run automatically. So an upgrade test is just two sequential Apply()
// calls on the same infrastructure with different version configs.
//
// YAML mutation between the two calls is done by unmarshaling the Terraform
// output into a generic map, updating the relevant fields, and re-marshaling —
// this avoids fragile string replacement and handles any extra fields the
// Terraform module injects (SANs, LB addresses, etc.).
//
// Upgrade paths tested
// ====================
//
//   TestUpgradeLegacyToModern
//     install:  MCR stable-25.0 / MKE 3.8.8   (legacy baseline)
//     upgrade:  MCR stable-29.2 / MKE 3.9.2   (modern target)
//     nodes:    rhel8/rocky8/ubuntu22 (same as TestLegacyCluster)
//
// These are the only hosts Launchpad CI provisions in the legacy matrix; they
// are also the most representative real-world upgrade path (customer sites
// running 3.8 that need to reach 3.9).

import (
	"fmt"
	"testing"

	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/Mirantis/launchpad/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// upgradeConfig pairs a base install smokeConfig with target upgrade versions.
type upgradeConfig struct {
	Base              smokeConfig
	UpgradeMCRChannel string
	UpgradeMKEVersion string
}

// runUpgradeTest provisions the cluster with Base versions, upgrades to the
// target versions, then resets and destroys.
func runUpgradeTest(t *testing.T, cfg upgradeConfig) {
	t.Helper()

	uTestId := test.GenerateRandomAlphaNumericString(5)
	name := fmt.Sprintf("smoke-%s-%s", cfg.Base.Name, uTestId)

	mkePassword := test.GenerateRandomAlphaNumericString(12)

	mkeConnect := map[string]interface{}{
		"username": "admin",
		"password": mkePassword,
		"insecure": true,
	}

	launchpad := map[string]interface{}{
		"drain":       false,
		"mcr_channel": cfg.Base.MCRChannel,
		"mke_version": cfg.Base.MKEVersion,
		"msr_version": cfg.Base.MSRVersion,
		"mke_connect": mkeConnect,
	}

	ngKeys := make([]string, 0, len(cfg.Base.Nodegroups))
	for k := range cfg.Base.Nodegroups {
		ngKeys = append(ngKeys, k)
	}

	subnets := map[string]interface{}{
		"main": map[string]interface{}{
			"cidr":       "172.31.0.0/17",
			"private":    false,
			"nodegroups": ngKeys,
		},
	}

	tempSSHKeyPathDir := t.TempDir()

	vars := map[string]interface{}{
		"name":              name,
		"aws":               awsConfig,
		"launchpad":         launchpad,
		"network":           networkConfig,
		"subnets":           subnets,
		"ssh_pk_location":   tempSSHKeyPathDir,
		"nodegroups":        cfg.Base.Nodegroups,
		"ssh_key_algorithm": cfg.Base.SSHKeyAlgorithm,
		"extra_tags": map[string]string{
			"launchpad-smoke-test":      "true",
			"launchpad-smoke-test-name": cfg.Base.Name,
		},
	}

	options := terraform.Options{
		TerraformDir: "../../examples/terraform/aws-simple",
		Vars:         vars,
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &options)
	defer terraform.Destroy(t, terraformOptions)

	if _, err := terraform.InitAndApplyE(t, terraformOptions); err != nil {
		t.Fatal(err)
	}

	baseYAML := terraform.Output(t, terraformOptions, "launchpad_yaml")

	// ── Step 1: install at base versions ─────────────────────────────────────
	t.Logf("installing base: MCR %s / MKE %s", cfg.Base.MCRChannel, cfg.Base.MKEVersion)

	baseProduct, err := config.ProductFromYAML([]byte(baseYAML))
	require.NoError(t, err, "parse base launchpad YAML")

	err = baseProduct.Apply(true, true, 3, true)
	require.NoError(t, err, "base install Apply()")

	// ── Step 2: build upgrade YAML ────────────────────────────────────────────
	// Unmarshal the Terraform-generated YAML into a generic map so we can
	// update version fields without disturbing host addresses, SANs, LB names,
	// or any other infrastructure-specific values the module injected.
	upgradeYAML, err := bumpVersions(baseYAML, cfg.UpgradeMCRChannel, cfg.UpgradeMKEVersion)
	require.NoError(t, err, "mutate YAML for upgrade")

	// ── Step 3: upgrade ───────────────────────────────────────────────────────
	t.Logf("upgrading to: MCR %s / MKE %s", cfg.UpgradeMCRChannel, cfg.UpgradeMKEVersion)

	upgradeProduct, err := config.ProductFromYAML([]byte(upgradeYAML))
	require.NoError(t, err, "parse upgrade launchpad YAML")

	err = upgradeProduct.Apply(true, true, 3, true)
	assert.NoError(t, err, "upgrade Apply()")

	// ── Step 4: reset (best-effort) ───────────────────────────────────────────
	// See smoke_test.go for rationale on non-fatal Reset().
	if err = upgradeProduct.Reset(); err != nil {
		t.Logf("WARN: product.Reset() failed (non-fatal): %v", err)
	}
}

// bumpVersions deserialises yamlStr, replaces spec.mcr.channel and
// spec.mke.version with the supplied values, and returns the re-serialised
// YAML. The rest of the document (hosts, SANs, LB addresses, flags, …) is
// preserved verbatim so the upgrade runs against the same infrastructure that
// was just provisioned.
func bumpVersions(yamlStr, mcrChannel, mkeVersion string) (string, error) {
	var doc map[interface{}]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &doc); err != nil {
		return "", fmt.Errorf("unmarshal cluster YAML: %w", err)
	}

	spec, ok := doc["spec"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("cluster YAML missing spec")
	}

	if mcr, ok := spec["mcr"].(map[interface{}]interface{}); ok {
		mcr["channel"] = mcrChannel
	} else {
		spec["mcr"] = map[interface{}]interface{}{"channel": mcrChannel}
	}

	if mke, ok := spec["mke"].(map[interface{}]interface{}); ok {
		mke["version"] = mkeVersion
	} else {
		return "", fmt.Errorf("cluster YAML missing spec.mke")
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("re-marshal upgraded YAML: %w", err)
	}
	return string(out), nil
}

// TestUpgradeLegacyToModern installs MKE 3.8.8 / MCR stable-25.0 on a
// legacy Linux matrix (rhel8, rocky8, ubuntu22) and then upgrades it to
// MKE 3.9.2 / MCR stable-29.2 in place. This is the primary real-world
// upgrade path for customers running the 3.8 stack.
func TestUpgradeLegacyToModern(t *testing.T) {
	runUpgradeTest(t, upgradeConfig{
		Base: smokeConfig{
			Name:            "upgrade",
			MCRChannel:      "stable-25.0",
			MKEVersion:      "3.8.8",
			MSRVersion:      "2.9.28",
			SSHKeyAlgorithm: "ed25519",
			Nodegroups: map[string]interface{}{
				"MngrRhel8":    test.Platforms["Rhel8"].GetManager(),
				"MngrRocky8":   test.Platforms["Rocky8"].GetManager(),
				"MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
				"WrkRhel8":     test.Platforms["Rhel8"].GetWorker(),
				"WrkRocky8":    test.Platforms["Rocky8"].GetWorker(),
				"WrkUbuntu22":  test.Platforms["Ubuntu22"].GetWorker(),
			},
		},
		UpgradeMCRChannel: "stable-29.2",
		UpgradeMKEVersion: "3.9.2",
	})
}
