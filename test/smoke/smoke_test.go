package smoke_test

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/Mirantis/launchpad/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

var awsConfig = map[string]interface{}{
	"region": "us-east-1",
}

var networkConfig = map[string]interface{}{
	"cidr":               "172.31.0.0/16",
	"enable_nat_gateway": false,
	"enable_vpn_gateway": false,
}

type smokeConfig struct {
	Name            string
	Nodegroups      map[string]interface{}
	MCRChannel      string
	MKEVersion      string
	MSRVersion      string
	SSHKeyAlgorithm string
}

// generateWindowsPassword returns a 20-character password satisfying Windows
// complexity requirements (upper, lower, digit, symbol).
func generateWindowsPassword(t *testing.T) string {
	t.Helper()
	const (
		upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lower   = "abcdefghijklmnopqrstuvwxyz"
		digits  = "0123456789"
		symbols = "!@#$%^&*"
		all     = upper + lower + digits + symbols
	)
	buf := make([]byte, 20)
	// Guarantee at least one of each required class at fixed positions.
	for i, charset := range []string{upper, lower, digits, symbols} {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			t.Fatalf("generateWindowsPassword: crypto/rand failed: %v", err)
		}
		buf[i] = charset[n.Int64()]
	}
	for i := 4; i < 20; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			t.Fatalf("generateWindowsPassword: crypto/rand failed: %v", err)
		}
		buf[i] = all[n.Int64()]
	}
	return string(buf)
}

func runSmokeTest(t *testing.T, cfg smokeConfig) {
	t.Helper()
	log.Printf("runSmokeTest: %s", cfg.Name)

	uTestId := test.GenerateRandomAlphaNumericString(5)
	name := fmt.Sprintf("smoke-%s-%s", cfg.Name, uTestId)

	mkePassword := test.GenerateRandomAlphaNumericString(12)

	mkeConnect := map[string]interface{}{
		"username": "admin",
		"password": mkePassword,
		"insecure": true,
	}

	launchpad := map[string]interface{}{
		"drain":       false,
		"mcr_channel": cfg.MCRChannel,
		"mke_version": cfg.MKEVersion,
		"msr_version": cfg.MSRVersion,
		"mke_connect": mkeConnect,
	}

	// Build subnet nodegroup list from nodegroup keys.
	ngKeys := make([]string, 0, len(cfg.Nodegroups))
	for k := range cfg.Nodegroups {
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
		"nodegroups":        cfg.Nodegroups,
		"ssh_key_algorithm": cfg.SSHKeyAlgorithm,
		"extra_tags": map[string]string{
			"launchpad-smoke-test":      "true",
			"launchpad-smoke-test-name": cfg.Name,
		},
	}

	// Detect windows nodegroups; pass windows_password if any present.
	hasWindows := false
	for _, ng := range cfg.Nodegroups {
		ngMap, ok := ng.(map[string]interface{})
		if !ok {
			continue
		}
		platform, _ := ngMap["platform"].(string)
		if strings.HasPrefix(platform, "windows_") {
			hasWindows = true
			break
		}
	}
	if hasWindows {
		vars["windows_password"] = generateWindowsPassword(t)
	}

	options := terraform.Options{
		TerraformDir: "../../examples/terraform/aws-simple",
		Vars:         vars,
	}

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &options)
	// Register destroy before apply so it runs even if apply partially succeeds
	// and then t.Fatal is called. t.Fatal calls runtime.Goexit which runs defers.
	defer terraform.Destroy(t, terraformOptions)
	// Registered after Destroy so it runs first (defers are LIFO): capture
	// EC2 console output for this stack's instances before they're torn
	// down, but only if the test already failed.
	defer dumpConsoleOutputOnFailure(t, name)
	if _, err := terraform.InitAndApplyE(t, terraformOptions); err != nil {
		t.Fatal(err)
	}

	mkeClusterConfig := terraform.Output(t, terraformOptions, "launchpad_yaml")

	product, err := config.ProductFromYAML([]byte(mkeClusterConfig))
	assert.NoError(t, err)

	err = product.Apply(true, true, 3, true)
	assert.NoError(t, err)

	// Reset is best-effort: the mirantis/ucp uninstall-ucp container has an
	// internal node-response timeout that fires before our go test timeout on
	// large or mixed-OS clusters (MKE 3.9.2 regression; Windows 2025 nodes).
	// Infrastructure is destroyed unconditionally by defer terraform.Destroy
	// above, so a Reset failure does not leave orphaned AWS resources.
	// Log the failure but do not fail the test on Reset errors.
	if err = product.Reset(); err != nil {
		t.Logf("WARN: product.Reset() failed (non-fatal): %v", err)
	}
}

// TestModernCluster exercises rhel9/ubuntu24/rocky9 managers and rhel9/sles15/ubuntu24/rocky9 workers
// with MCR stable-29.2 and MKE 3.9.2.
func TestModernCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		Name:            "modern",
		MCRChannel:      "stable-29.2",
		MKEVersion:      "3.9.2",
		MSRVersion:      "3.1.18",
		SSHKeyAlgorithm: "ed25519",
		Nodegroups: map[string]interface{}{
			"MngrRhel9":    test.Platforms["Rhel9"].GetManager(),
			"MngrUbuntu24": test.Platforms["Ubuntu24"].GetManager(),
			"MngrRocky9":   test.Platforms["Rocky9"].GetManager(),
			"WrkRhel9":     test.Platforms["Rhel9"].GetWorker(),
			"WrkSles15":    test.Platforms["Sles15"].GetWorker(),
			"WrkUbuntu24":  test.Platforms["Ubuntu24"].GetWorker(),
			"WrkRocky9":    test.Platforms["Rocky9"].GetWorker(),
		},
	})
}

// TestCuttingEdgeCluster exercises rhel10/rocky10/ubuntu26 managers and workers
// with sles16 as an additional worker, using the latest MCR and MKE versions.
// The rhel_10/rocky_10 platform keys are available in
// terraform-mirantis-provision-aws >= v0.1.7 (pinned in examples/terraform/aws-simple).
//
// NOTE: as of 2026-07-22, MCR has not published Docker EE packages for either
// SLES 16 (repos.mirantis.com/sles/ only has 12/12.3/15 - no 16 directory) or
// Ubuntu 26.04 "resolute" (repos.mirantis.com/ubuntu/dists/ only has
// trusty/xenial/bionic/focal/jammy/noble - no resolute directory). Both nodes
// are expected to fail at the Install MCR phase until MCR ships support; see
// also the RHEL10/Rocky10 MCR-daemon-start gap (missing xt_* kernel modules
// break iptables, and MCR's nftables backend is incompatible with swarm mode)
// tracked separately.
func TestCuttingEdgeCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		// AWS LB/target-group names are capped at 32 chars: the stack name is
		// "smoke-{Name}-{5-char-random}" and Terraform appends suffixes like
		// "-mke-kube" (9 chars), capping len(Name) at 11. Hence "cuttingedge"
		// rather than "cutting-edge" (12 chars).
		Name:            "cuttingedge",
		MCRChannel:      "stable-29.4",
		MKEVersion:      "3.9.2",
		MSRVersion:      "3.1.18",
		SSHKeyAlgorithm: "ed25519",
		Nodegroups: map[string]interface{}{
			"MngrRhel10":   test.Platforms["Rhel10"].GetManager(),
			"MngrRocky10":  test.Platforms["Rocky10"].GetManager(),
			"MngrUbuntu26": test.Platforms["Ubuntu26"].GetManager(),
			"WrkRhel10":    test.Platforms["Rhel10"].GetWorker(),
			"WrkRocky10":   test.Platforms["Rocky10"].GetWorker(),
			"WrkSles16":    test.Platforms["Sles16"].GetWorker(),
			"WrkUbuntu26":  test.Platforms["Ubuntu26"].GetWorker(),
		},
	})
}

// TestLegacyCluster exercises rhel8/rocky8/ubuntu22 managers and workers
// with MCR stable-25.0 and MKE 3.8.8. sles12 was tried as a worker here but
// launchpad's Validate Hosts phase fails on it (hostname --all-ip-addresses
// is unsupported on SLES 12's toolchain) -- see PRODENG-3588.
func TestLegacyCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		Name:            "legacy",
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
	})
}

// TestWindowsCluster exercises ubuntu24 manager and windows_2019/2022/2025 workers
// with MCR stable-25.0 and MKE 3.8.8. Uses RSA keypair (required for Windows password retrieval).
func TestWindowsCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		Name:            "windows",
		MCRChannel:      "stable-25.0",
		MKEVersion:      "3.8.8",
		MSRVersion:      "2.9.28",
		SSHKeyAlgorithm: "rsa",
		Nodegroups: map[string]interface{}{
			"MngrUbuntu24": test.Platforms["Ubuntu24"].GetManager(),
			"WrkWin2019":   test.Platforms["Windows2019"].GetWorker(),
			"WrkWin2022":   test.Platforms["Windows2022"].GetWorker(),
			"WrkWin2025":   test.Platforms["Windows2025"].GetWorker(),
		},
	})
}

// TestFIPSCluster exercises an ubuntu_22.04_fips manager and a windows_2022 worker
// with MCR stable-29.2.1/fips and MKE 3.9.2. Validates that the Windows
// installer correctly resolves a versioned FIPS artifact from the channel
// index rather than attempting the non-existent docker-latest+fips.zip.
// Uses RSA keypair (required for Windows password retrieval).
func TestFIPSCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		Name:            "fips",
		MCRChannel:      "stable-29.2.1/fips",
		MKEVersion:      "3.9.2",
		MSRVersion:      "3.1.18",
		SSHKeyAlgorithm: "rsa",
		Nodegroups: map[string]interface{}{
			"MngrUbuntu22FIPS": test.Platforms["Ubuntu22FIPS"].GetManager(),
			"WrkWin2025":       test.Platforms["Windows2025"].GetWorker(),
		},
	})
}
