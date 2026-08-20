package smoke_test

// Swarm overlay address pool smoke coverage for PRODENG-3642.
//
// A swarm's --default-addr-pool is fixed when the swarm is created: it is a
// field of docker's swarm InitRequest but not of the Spec that
// "docker swarm update" mutates. Launchpad therefore validates --pod-cidr
// against the pool the running swarm actually uses, and only against
// spec.mcr.swarmInstallFlags when no swarm exists yet.
//
// Two things can only be demonstrated against a real cluster, and are covered
// here rather than in pkg/product/mke/phase unit tests:
//
//   - a conflict is fatal while it is still avoidable, i.e. before the swarm
//     exists, and configuring a pool genuinely resolves it
//   - swarm init really does apply the configured pool, so a later apply
//     discovers that pool rather than docker's default
//
// The warn-not-fail behaviour on an existing swarm is asserted in
// upgrade_test.go, where CI already applies twice against one cluster.
//
// This test is label-gated and is not part of the per-merge matrix: it spends a
// stack purely on validation logic that changes rarely.
//
// Provisioning is duplicated from runSmokeTest rather than shared. That is
// deliberate: runSmokeTest's "defer terraform.Destroy" ordering is what keeps
// failed runs from leaking VPCs (PRODENG-3631), and reshaping it to be reusable
// would put the teardown of four existing smoke tests at risk to save ~40 lines.

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/Mirantis/launchpad/pkg/product"
	"github.com/Mirantis/launchpad/pkg/product/mke"
	"github.com/Mirantis/launchpad/pkg/swarm"
	"github.com/Mirantis/launchpad/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	// poolOutsideVPC is a pool the stack's 172.31.0.0/16 VPC does not contain,
	// so configuring it cannot break host networking, and which docker's
	// 10.0.0.0/8 default does contain, so a swarm created with it is
	// distinguishable from a swarm created without one.
	poolOutsideVPC = "10.99.0.0/16"

	// podCIDRConflictingWithDefault is the customer's value from PRODENG-3642.
	// It overlaps docker's default pool and not poolOutsideVPC.
	podCIDRConflictingWithDefault = "10.244.0.0/16"
)

// TestSwarmAddrPoolCluster covers the install side of PRODENG-3642 on one stack:
// the conflict is refused while it is still preventable, configuring a pool
// resolves it, and the pool is genuinely in force afterwards.
func TestSwarmAddrPoolCluster(t *testing.T) {
	cfg := smokeConfig{
		Name:            "swarmpool",
		MCRChannel:      "stable-29.2",
		MKEVersion:      "3.9.2",
		MSRVersion:      "3.1.18",
		SSHKeyAlgorithm: "ed25519",
		// Two hosts is the smallest cluster that still exercises a real swarm.
		Nodegroups: map[string]interface{}{
			"MngrUbuntu24": test.Platforms["Ubuntu24"].GetManager(),
			"WrkUbuntu24":  test.Platforms["Ubuntu24"].GetWorker(),
		},
	}

	baseYAML := provisionSwarmPoolStack(t, cfg)

	// ── Step 1: the conflict must be refused, not worked around ──────────────
	// No pool is configured, so validation compares against docker's default
	// and must fail. This costs nothing beyond a connect and fact-gather: the
	// run aborts in Validate Facts, long before MKE is installed, so the broken
	// network state the check exists to prevent is never created.
	conflictingYAML, err := setSwarmPoolAndPodCIDR(baseYAML, "", podCIDRConflictingWithDefault)
	require.NoError(t, err, "inject conflicting pod CIDR")

	conflictingProduct, err := config.ProductFromYAML([]byte(conflictingYAML))
	require.NoError(t, err, "parse conflicting launchpad YAML")

	err = conflictingProduct.Apply(true, true, 3, true)
	require.Error(t, err, "a pod CIDR overlapping the swarm pool must fail while no swarm exists")
	assert.Contains(t, err.Error(), "overlaps with the Swarm overlay address pool",
		"failure must name the overlap rather than surface as something else")

	// ── Step 2: configuring a pool must resolve it ────────────────────────────
	// swarm init has not run yet, so swarmInstallFlags is still the pool that
	// will be applied -- which is exactly why this is the one situation where
	// launchpad's suggested remedy works.
	installYAML, err := setSwarmPoolAndPodCIDR(baseYAML, poolOutsideVPC, podCIDRConflictingWithDefault)
	require.NoError(t, err, "inject pool and pod CIDR")

	installProduct, err := config.ProductFromYAML([]byte(installYAML))
	require.NoError(t, err, "parse install launchpad YAML")

	installLogs := captureLaunchpadLogs(t, func() {
		err = installProduct.Apply(true, true, 3, true)
	})
	require.NoError(t, err, "configuring a non-overlapping pool must allow the install to proceed")
	assert.NotContains(t, installLogs, "of the existing swarm",
		"a fresh install has no existing swarm to report against")

	// ── Step 3: the configured pool must actually be in force ────────────────
	// GatherFacts only reads the pool when a swarm already exists, so a second
	// apply is what observes the result of step 2. Anything other than
	// poolOutsideVPC below means swarm init ignored the flag.
	verifyProduct, err := config.ProductFromYAML([]byte(installYAML))
	require.NoError(t, err, "re-parse install launchpad YAML")

	verifyLogs := captureLaunchpadLogs(t, func() {
		err = verifyProduct.Apply(true, true, 3, true)
	})

	// This apply is a means of triggering discovery, not the behaviour under
	// test. Gather Facts runs early, so the assertions below are already
	// decided by the time anything later in the run can fail -- and those later
	// phases install packages from public mirrors, which is a flake surface this
	// test has no reason to inherit. A genuine failure to discover the pool
	// still fails, on the assertion that actually checks it.
	if err != nil {
		t.Logf("re-apply did not run to completion; this does not affect the "+
			"assertions below, which only need the Gather Facts phase: %v", err)
	}

	assert.Equal(t, []string{poolOutsideVPC}, discoveredAddrPool(t, verifyProduct),
		"swarm init must have applied the configured pool")
	assert.NotContains(t, verifyLogs, "has no effect here",
		"config matches the running swarm, so nothing has diverged")
	assert.NotContains(t, verifyLogs, "overlaps the overlay address pool",
		"the configured pool does not overlap the pod CIDR")

	if err = verifyProduct.Reset(); err != nil {
		t.Logf("WARN: product.Reset() failed (non-fatal): %v", err)
	}
}

// provisionSwarmPoolStack brings up infrastructure and returns the generated
// cluster YAML. Teardown is registered before apply so it still runs when a
// later require() aborts the test.
func provisionSwarmPoolStack(t *testing.T, cfg smokeConfig) string {
	t.Helper()

	name := fmt.Sprintf("smoke-%s-%s", cfg.Name, test.GenerateRandomAlphaNumericString(5))

	ngKeys := make([]string, 0, len(cfg.Nodegroups))
	for k := range cfg.Nodegroups {
		ngKeys = append(ngKeys, k)
	}

	vars := map[string]interface{}{
		"name": name,
		"aws":  awsConfig,
		"launchpad": map[string]interface{}{
			"drain":       false,
			"mcr_channel": cfg.MCRChannel,
			"mke_version": cfg.MKEVersion,
			"msr_version": cfg.MSRVersion,
			"mke_connect": map[string]interface{}{
				"username": "admin",
				"password": test.GenerateRandomAlphaNumericString(12),
				"insecure": true,
			},
		},
		"network": networkConfig,
		"subnets": map[string]interface{}{
			"main": map[string]interface{}{
				"cidr":       "172.31.0.0/17",
				"private":    false,
				"nodegroups": ngKeys,
			},
		},
		"ssh_pk_location":   t.TempDir(),
		"nodegroups":        cfg.Nodegroups,
		"ssh_key_algorithm": cfg.SSHKeyAlgorithm,
		"extra_tags": map[string]string{
			"launchpad-smoke-test":      "true",
			"launchpad-smoke-test-name": cfg.Name,
		},
	}

	options := terraform.Options{
		TerraformDir: "../../examples/terraform/aws-simple",
		Vars:         vars,
	}
	terraformOptions := terraform.WithDefaultRetryableErrors(t, &options)

	// Registered before apply so a partial apply is still torn down, and via
	// t.Cleanup so it survives the runtime.Goexit that require() triggers.
	// Cleanups are LIFO, so the console capture registered next runs first,
	// while the instances still exist.
	t.Cleanup(func() { terraform.Destroy(t, terraformOptions) })
	t.Cleanup(func() { dumpConsoleOutputOnFailure(t, name) })

	if _, err := terraform.InitAndApplyE(t, terraformOptions); err != nil {
		t.Fatal(err)
	}

	return terraform.Output(t, terraformOptions, "launchpad_yaml")
}

// setSwarmPoolAndPodCIDR rewrites spec.mcr.swarmInstallFlags and
// spec.mke.installFlags. An empty value leaves that setting alone. The document
// is round-tripped through a generic map, as bumpVersions does, so
// infrastructure values the Terraform module injected survive untouched.
func setSwarmPoolAndPodCIDR(yamlStr, addrPool, podCIDR string) (string, error) {
	var doc map[interface{}]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &doc); err != nil {
		return "", fmt.Errorf("unmarshal cluster YAML: %w", err)
	}

	spec, ok := doc["spec"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("cluster YAML missing spec")
	}

	if addrPool != "" {
		mcr, ok := spec["mcr"].(map[interface{}]interface{})
		if !ok {
			mcr = map[interface{}]interface{}{}
			spec["mcr"] = mcr
		}
		mcr["swarmInstallFlags"] = []interface{}{"--default-addr-pool=" + addrPool}
	}

	if podCIDR != "" {
		mkeSpec, ok := spec["mke"].(map[interface{}]interface{})
		if !ok {
			return "", fmt.Errorf("cluster YAML missing spec.mke")
		}

		// Drop any --pod-cidr the module already set, so the value under test is
		// unambiguous rather than appended alongside another one.
		existing, _ := mkeSpec["installFlags"].([]interface{})
		flags := make([]interface{}, 0, len(existing)+1)
		for _, f := range existing {
			if s, ok := f.(string); ok && strings.HasPrefix(s, "--pod-cidr") {
				continue
			}
			flags = append(flags, f)
		}
		mkeSpec["installFlags"] = append(flags, "--pod-cidr="+podCIDR)
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("re-marshal cluster YAML: %w", err)
	}

	return string(out), nil
}

// discoveredAddrPool returns the pool GatherFacts read from the swarm leader.
// It is empty unless a swarm existed when the apply started.
func discoveredAddrPool(t *testing.T, p product.Product) []string {
	t.Helper()

	mkeProduct, ok := p.(*mke.MKE)
	require.True(t, ok, "expected an MKE product, got %T", p)
	require.NotNil(t, mkeProduct.ClusterConfig.Spec.MCR.Metadata, "MCR metadata must be populated")

	return mkeProduct.ClusterConfig.Spec.MCR.Metadata.SwarmDefaultAddrPool
}

// captureLaunchpadLogs collects what fn logs while leaving the original output
// in place, so CI still shows the apply as it happens.
func captureLaunchpadLogs(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	logger := logrus.StandardLogger()
	original := logger.Out
	logger.SetOutput(io.MultiWriter(original, &buf))

	t.Cleanup(func() { logger.SetOutput(original) })

	fn()

	return buf.String()
}

// assertExistingSwarmPoolBehaviour checks the contract that applies once a swarm
// exists: the configured pool is inert and said to be, an overlap with the live
// pool is reported, and neither condition fails the run.
func assertExistingSwarmPoolBehaviour(t *testing.T, p product.Product, logs string) {
	t.Helper()

	assert.Contains(t, logs, "--default-addr-pool "+poolOutsideVPC,
		"a configured pool the running swarm does not use must be reported, naming both values")
	assert.Contains(t, logs, "has no effect here",
		"the report must state that the setting does nothing here")

	assert.Contains(t, logs, "overlaps the overlay address pool "+swarm.DefaultAddrPoolFallback+" of the existing swarm",
		"the overlap must be reported against the live pool, not the configured one")

	// The warning above claims the configured pool was not applied. Confirm that
	// is true rather than trusting the message.
	assert.Equal(t, []string{swarm.DefaultAddrPoolFallback}, discoveredAddrPool(t, p),
		"the running swarm must still use the pool it was created with")
}
