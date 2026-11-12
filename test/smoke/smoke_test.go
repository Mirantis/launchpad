package smoke_test

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/Mirantis/launchpad/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// smokeRegion is the AWS region every smoke stack is provisioned in, and the
// region console-output capture queries. us-west-2 allows 5 VPCs and five
// smoke jobs run per push to main, so stacks fail at `terraform apply` with
// VpcLimitExceeded; us-east-1 allows 50. The c6a.xlarge exhaustion that
// prompted the move away from us-east-1 (runs 39792899324, 39793016128,
// 39793337401) came from every nodegroup landing in one AZ, because the
// provision module derives AZs as slice(available_azs, 0, len(subnets)) and
// the tests declared a single subnet. The partitioning below fixes that
// directly rather than by changing region.
const smokeRegion = "us-east-1"

var awsConfig = map[string]interface{}{
	"region": smokeRegion,
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
	MCRRepoURL      string // optional; empty uses launchpad.tf's public-repo default
	MKEVersion      string
	MSRVersion      string
	SSHKeyAlgorithm string
}

// generateWindowsPassword returns a 20-character password satisfying Windows
// complexity requirements (upper, lower, digit, symbol).
func generateWindowsPassword(t *testing.T) string {
	t.Helper()
	const (
		upper  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lower  = "abcdefghijklmnopqrstuvwxyz"
		digits = "0123456789"
		// TEMPORARY WORKAROUND, see PRODENG-3751: "$" is excluded here only
		// to unblock this branch's CI. launchpad.tf embeds this password
		// directly into the launchpad_yaml output (password/adminPassword
		// fields), which launchpad's config loader then runs through
		// envsubst (pkg/config/config.go); an unescaped "$word" is now
		// correctly treated as an environment variable reference and fails
		// config loading if unset (PRODENG-3751 made this fail loudly
		// instead of silently corrupting the password, which is correct).
		// The proper fix is to escape "$" as "$$" where launchpad.tf embeds
		// the password into that YAML, so a literal "$" -- legitimate in a
		// real password -- keeps working end to end; do that and restore
		// "$" here instead of leaving it excluded.
		symbols = "!@#%^&*"
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
		// PRODENG-3924: a stock Windows host refuses the signed install
		// script under -ExecutionPolicy AllSigned, so every Windows host
		// fails at Install MCR before any launchpad behaviour is reached.
		// The suites opt out until that is understood; the reason the host
		// refuses is captured by installerDiagnostics in windows.go.
		"mcr_install_url_windows_allow_unsigned": true,
	}
	if cfg.MCRRepoURL != "" {
		launchpad["mcr_repo_url"] = cfg.MCRRepoURL
	}

	// Build subnet nodegroup list from nodegroup keys.
	ngKeys := make([]string, 0, len(cfg.Nodegroups))
	for k := range cfg.Nodegroups {
		ngKeys = append(ngKeys, k)
	}
	// Sorted because Go randomises map iteration order: without this the
	// round-robin below would place a given nodegroup in a different AZ on
	// every run, making an AZ capacity failure unreproducible.
	sort.Strings(ngKeys)

	// One subnet per AZ, each nodegroup in exactly one of them. The provision
	// module derives AZs as slice(available_azs, 0, len(subnets)), so the
	// subnet count is the AZ count. Spreading the nodegroups is what stops one
	// AZ's instance-type capacity failing the whole stack. The CIDRs subdivide
	// the /17 the single subnet used, leaving the VPC's upper half free.
	subnetCIDRs := []string{"172.31.0.0/19", "172.31.32.0/19", "172.31.64.0/19"}
	if len(ngKeys) < len(subnetCIDRs) {
		subnetCIDRs = subnetCIDRs[:len(ngKeys)]
	}

	subnetNgs := make([][]string, len(subnetCIDRs))
	for i, k := range ngKeys {
		n := i % len(subnetCIDRs)
		subnetNgs[n] = append(subnetNgs[n], k)
	}

	subnets := make(map[string]interface{}, len(subnetCIDRs))
	for i, cidr := range subnetCIDRs {
		subnets[fmt.Sprintf("main%d", i)] = map[string]interface{}{
			"cidr":       cidr,
			"private":    false,
			"nodegroups": subnetNgs[i],
		}
	}

	tempSSHKeyPathDir := t.TempDir()

	vars := map[string]interface{}{
		"name":              name,
		"aws":               awsConfig,
		"admin_cidrs":       test.AdminCIDRs(t),
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

	// launchpad nests the MKE admin password, so it travels by TF_VAR_* rather
	// than as a "-var" argument. See test.TerraformOptions.
	secretVars := map[string]interface{}{
		"launchpad": launchpad,
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
	windowsPassword := ""
	if hasWindows {
		windowsPassword = generateWindowsPassword(t)
		secretVars["windows_password"] = windowsPassword
	}

	options := test.TerraformOptions(t, "../../examples/terraform/aws-simple", vars, secretVars)

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

	mkeClusterConfig := test.OutputSecret(t, terraformOptions, "launchpad_yaml")

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

// TestCuttingEdgeCluster exercises rhel10/rocky10/ubuntu26 managers and
// workers, using the latest MCR and MKE versions. The rhel_10/rocky_10
// platform keys are available in terraform-mirantis-provision-aws >= v0.1.7
// (pinned to 0.1.8 in examples/terraform/aws-simple).
//
// MCR package availability, checked 2026-11-03:
//   - rhel/10 (also used by rocky10): stable-29.2 through stable-29.8.0
//   - ubuntu resolute: stable-29.8.0 and test-29.8.0 only, hence the channel
//   - sles/16: no directory under repos.mirantis.com/sles/ (12, 12.3, 15
//     only), so a sles16 worker cannot install MCR and is not in the matrix
//
// Separately tracked: the RHEL10/Rocky10 MCR-daemon-start gap (missing xt_*
// kernel modules break iptables, and MCR's nftables backend is incompatible
// with swarm mode).
func TestCuttingEdgeCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		// AWS LB/target-group names are capped at 32 chars: the stack name is
		// "smoke-{Name}-{5-char-random}" and Terraform appends suffixes like
		// "-mke-kube" (9 chars), capping len(Name) at 11. Hence "cuttingedge"
		// rather than "cutting-edge" (12 chars).
		Name:            "cuttingedge",
		MCRChannel:      "stable-29.8.0",
		MKEVersion:      "3.9.7",
		MSRVersion:      "3.1.18",
		SSHKeyAlgorithm: "ed25519",
		Nodegroups: map[string]interface{}{
			"MngrRhel10":   test.Platforms["Rhel10"].GetManager(),
			"MngrRocky10":  test.Platforms["Rocky10"].GetManager(),
			"MngrUbuntu26": test.Platforms["Ubuntu26"].GetManager(),
			"WrkRhel10":    test.Platforms["Rhel10"].GetWorker(),
			"WrkRocky10":   test.Platforms["Rocky10"].GetWorker(),
			"WrkUbuntu26":  test.Platforms["Ubuntu26"].GetWorker(),
		},
	})
}

// TestLegacyCluster exercises rhel8/rocky8/ubuntu22 managers and workers
// with MCR stable-25.0 and MKE 3.8.8. sles12 was tried as a worker here but
// launchpad's Validate Hosts phase fails on it (hostname --all-ip-addresses
// is unsupported on SLES 12's toolchain) -- see PRODENG-3588.
//
// This is the only suite with an msr-role nodegroup, and so the only one
// that installs MSR: examples/terraform/aws-simple/launchpad.tf derives
// local.has_msr (:207) from whether any nodegroup has role "msr", and
// without one it emits "kind: mke" and omits the spec.msr block entirely,
// leaving MSRVersion inert. MSR 2.9.28 is compatible with MKE 3.8.x and
// MCR 23.x/25.x, which is what this suite installs.
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
			"MsrUbuntu22":  test.Platforms["Ubuntu22"].GetMSR(),
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

// TestMSRCluster is the only suite that installs MSR.
//
// It exists because MSR cannot ride along with the other suites. test/
// platforms.go has always defined GetMSR(), but nothing called it, so
// launchpad.tf's has_msr (:207, counting msr-role nodegroups) was always
// false: every suite rendered `kind: mke`, the spec.msr block was skipped,
// and the MSRVersion each one declares was discarded at the last gate. No
// smoke test had ever installed MSR.
//
// The versions are the legacy stack, not the modern one, and that is forced.
// Measured on CI run 43022186758, MSR 2.9.28 against MCR 29.2:
//
//	FATA Failed to get new conv client: Docker version check failed:
//	client version 1.24 is too old. Minimum supported API version is 1.40
//
// The MSR 2.x bootstrapper speaks Docker API 1.24 and MCR 29 requires 1.40,
// so MSR 2.x cannot install on a modern cluster at all. MCR stable-25.0 still
// accepts 1.24. MSR 3.x is not an escape either: launchpad.tf renders MSR 2.x
// config only (`kind: mke+msr`, --dtr-external-url, --ucp-insecure-tls) and
// this launchpad has no MSR 3 support -- no msr3 kind anywhere in pkg/, and
// the MSR phases are DTR-era.
//
// Hence its own `smoke-msr` label rather than a nodegroup added to an
// existing suite: the only stack MSR runs on is one no other suite installs
// by default, and an MSR node costs an instance and an ingress per run.
func TestMSRCluster(t *testing.T) {
	runSmokeTest(t, smokeConfig{
		Name:            "msr",
		MCRChannel:      "stable-25.0",
		MKEVersion:      "3.8.8",
		MSRVersion:      "2.9.28",
		SSHKeyAlgorithm: "ed25519",
		Nodegroups: map[string]interface{}{
			"MngrRhel8":   test.Platforms["Rhel8"].GetManager(),
			"WrkUbuntu22": test.Platforms["Ubuntu22"].GetWorker(),
			"MsrUbuntu22": test.Platforms["Ubuntu22"].GetMSR(),
		},
	})
}
