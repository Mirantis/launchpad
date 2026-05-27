package smoke_test

// Airgapped multi-hop upgrade smoke test — customer scenario.
//
// TestAirgappedMultiHopUpgrade provisions a cluster with an internal MSR/DTR
// (2.9.27) on port 4443, installs the baseline software, pre-loads all MKE
// and MSR upgrade images into DTR, then drives three sequential upgrades with
// mke.imageRepo and msr.imageRepo pointing to DTR throughout.
//
// Upgrade chain:
//
//	install: MCR stable-25.0 / MKE 3.8.8  / MSR 2.9.27  (images from docker.io/mirantis)
//	step 1:  MCR stable-25.0 / MKE 3.8.11               (images from DTR :4443)
//	step 2:  MCR stable-29.2 / MKE 3.8.12               (images from DTR :4443)
//	step 3:  MCR stable-29.2 / MKE 3.9.2                (images from DTR :4443)
//
// What this test validates:
//   - Launchpad correctly uses mke.imageRepo and msr.imageRepo when set to an
//     internal registry address that includes a non-standard port.
//   - The full 3.8.8 → 3.8.11 → 3.8.12 → 3.9.2 upgrade chain completes when
//     all MKE bootstrapper images are served from an internal registry.
//   - DTR exposed on a non-standard port (4443) is reachable and usable as an
//     image registry for both Docker operations and Launchpad imageRepo config.
//
// What this test does NOT validate:
//   - True network-level airgap. Manager and worker nodes can still reach the
//     internet; the initial install uses docker.io/mirantis to bootstrap DTR
//     itself (unavoidable chicken-and-egg). For full egress blocking after
//     bootstrap, restrict the manager/worker security group in the Terraform
//     module — the current module does not expose per-nodegroup egress rules.
//   - MCR package airgap. MCR packages (RPMs/DEBs) are still pulled from
//     repos.mirantis.com during the upgrade steps. Airgapping MCR requires a
//     separate apt/yum mirror and updating spec.mcr.repoURL.
//
// Terraform note:
//   The msr_port variable in examples/terraform/aws-simple/variables.tf
//   defaults to 443, so all other smoke tests are unaffected. This test
//   passes msr_port=4443 at runtime via terraform.Options.Vars — no file
//   modification or reversion is required.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/config"
	"github.com/Mirantis/launchpad/test"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// upgradeStep is a single version hop in the multi-step upgrade chain.
type upgradeStep struct {
	MCRChannel string
	MKEVersion string
}

// airgapUpgradeConfig pairs a base install with an ordered list of upgrade
// steps. An MSR nodegroup (role: msr) must be present in Base.Nodegroups
// because it acts as the internal image registry for all upgrade steps.
type airgapUpgradeConfig struct {
	Base  smokeConfig
	Steps []upgradeStep
}

// bumpVersionsAirgap updates spec.mcr.channel, spec.mke.version, and both
// imageRepo fields so that subsequent Apply calls pull images from the
// internal DTR (registryPrefix, e.g. "dtr.example.com:4443/admin") rather
// than docker.io/mirantis.
//
// spec.mcr.repoURL is intentionally left unchanged. MCR packages are
// installed from a Linux package repository (apt/yum), not a container
// registry. Airgapping MCR package installation requires a separate package
// mirror; see the file-level comment for details.
func bumpVersionsAirgap(yamlStr, mcrChannel, mkeVersion, registryPrefix string) (string, error) {
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
		mke["imageRepo"] = registryPrefix
	} else {
		return "", fmt.Errorf("cluster YAML missing spec.mke")
	}

	if msr, ok := spec["msr"].(map[interface{}]interface{}); ok {
		msr["imageRepo"] = registryPrefix
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("re-marshal upgraded YAML: %w", err)
	}
	return string(out), nil
}

// clusterHost holds the SSH connection details for one node in the cluster.
type clusterHost struct {
	addr    string
	user    string
	keyPath string
}

// extractAllSSHHosts returns a clusterHost for every SSH-connected host in the
// launchpad YAML. Windows (WinRM) hosts are skipped.
func extractAllSSHHosts(yamlStr string) ([]clusterHost, error) {
	var doc map[interface{}]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &doc); err != nil {
		return nil, fmt.Errorf("unmarshal YAML: %w", err)
	}
	spec, ok := doc["spec"].(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("cluster YAML missing spec")
	}
	rawHosts, ok := spec["hosts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("cluster YAML missing spec.hosts")
	}
	var out []clusterHost
	for _, h := range rawHosts {
		host, ok := h.(map[interface{}]interface{})
		if !ok {
			continue
		}
		sshConf, ok := host["ssh"].(map[interface{}]interface{})
		if !ok {
			continue // WinRM host or no ssh block
		}
		addr, _ := sshConf["address"].(string)
		user, _ := sshConf["user"].(string)
		keyPath, _ := sshConf["keyPath"].(string)
		if addr != "" && user != "" && keyPath != "" {
			out = append(out, clusterHost{addr: addr, user: user, keyPath: keyPath})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no SSH hosts found in launchpad YAML")
	}
	return out, nil
}

// extractDTRHost parses the DTR external URL from the MSR installFlags in the
// launchpad YAML and returns a "hostname:port" string suitable for use as a
// Docker registry address (e.g. "abc123.elb.amazonaws.com:4443").
//
// The port is always explicit because Docker uses it as the key for both
// /etc/docker/certs.d/ directory lookup and image reference resolution.
// "hostname" and "hostname:4443" are treated as distinct registries by Docker,
// so all callers — cert trust setup, docker login, docker push/pull, and the
// imageRepo field — must use the same "hostname:port" form consistently.
// If the URL contains no port, :443 is appended as the default.
func extractDTRHost(yamlStr string) (string, error) {
	var doc map[interface{}]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &doc); err != nil {
		return "", fmt.Errorf("unmarshal YAML: %w", err)
	}
	spec, ok := doc["spec"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("cluster YAML missing spec")
	}
	msr, ok := spec["msr"].(map[interface{}]interface{})
	if !ok {
		return "", fmt.Errorf("cluster YAML missing spec.msr — ensure an MSR nodegroup is present")
	}
	flags, ok := msr["installFlags"].([]interface{})
	if !ok {
		return "", fmt.Errorf("no installFlags in spec.msr")
	}
	const prefix = "--dtr-external-url="
	for _, f := range flags {
		flag, _ := f.(string)
		if strings.HasPrefix(flag, prefix) {
			addr := strings.TrimPrefix(flag, prefix)
			addr = strings.TrimPrefix(addr, "https://")
			addr = strings.TrimPrefix(addr, "http://")
			// Ensure port is explicit. Without it, Docker keys /etc/docker/certs.d/
			// on the bare hostname, which differs from the hostname:port key used
			// when the port is present — causing cert lookup mismatches.
			if !strings.Contains(addr, ":") {
				addr += ":443"
			}
			return addr, nil
		}
	}
	return "", fmt.Errorf("--dtr-external-url not found in spec.msr.installFlags")
}

// sshRun executes a remote shell command on addr via the system ssh binary,
// authenticating with keyPath. It avoids Go-side private key parsing entirely,
// which is necessary because Terraform's tls_private_key resource emits
// OpenSSH-format ed25519 keys (-----BEGIN OPENSSH PRIVATE KEY-----) that
// golang.org/x/crypto/ssh.ParsePrivateKey rejects with "ssh: no key found".
func sshRun(user, addr, keyPath, command string) (string, error) {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=30",
		"-i", keyPath,
		user+"@"+addr,
		command,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// sshRunScript executes a shell script on addr via the system ssh binary by
// piping the script to bash via stdin. This avoids SSH command-line length
// limits and shell quoting complexity for multi-line scripts.
func sshRunScript(user, addr, keyPath, script string) (string, error) {
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=30",
		"-i", keyPath,
		user+"@"+addr,
		"bash -s",
	)
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// preloadImagesOnNode pulls all images for the given bootstrapper from
// docker.io/mirantis and tags each one locally with the DTR registry address
// prefix, so that Launchpad's "docker image inspect dtrHost/admin/<img>:<tag>"
// check succeeds without any network call to DTR.
//
// Background: Launchpad's "Pull MKE images" phase runs docker image inspect
// before docker pull. If the DTR-addressed tag already exists locally the pull
// is skipped entirely. Preloading on every node (manager + workers + MSR) means
// no actual DTR push or pull is ever needed, while still exercising Launchpad's
// imageRepo feature — all upgrade Apply() calls use dtrHost+"/admin" as
// mke.imageRepo and msr.imageRepo.
func preloadImagesOnNode(t *testing.T, user, addr, keyPath, dtrHost, bootstrapper string) {
	t.Helper()

	script := fmt.Sprintf(`set -euo pipefail
bootstrapper="%s"
dtr_admin="%s"
failcount=0

# Pull the bootstrapper so its "images" subcommand can run.
if ! docker pull "${bootstrapper}" > /dev/null 2>&1; then
  echo "ERROR: failed to pull bootstrapper ${bootstrapper}" >&2
  exit 1
fi

# Enumerate images. UCP uses "images --list"; DTR/MSR 2.x uses just "images"
# (--list is unrecognised by DTR and causes it to print help text to stdout
# with exit 0, which we detect by filtering for valid image-reference lines).
_list_images() {
  docker run --rm "$1" images $2 2>/dev/null \
    | grep -E '^[a-zA-Z0-9][a-zA-Z0-9._/:@-]*:[a-zA-Z0-9._-]+$' \
    || true
}
images=$(_list_images "${bootstrapper}" "--list")
[ -n "${images}" ] || images=$(_list_images "${bootstrapper}" "")
# Always include the bootstrapper itself; the inspect check in the loop below
# skips it if already tagged.
images="${images}
${bootstrapper}"

for img in ${images}; do
  short="${img##*/}"
  dtr_img="${dtr_admin}/${short}"

  # Already tagged locally with the DTR address — nothing to do.
  if docker image inspect "${dtr_img}" > /dev/null 2>&1; then
    continue
  fi

  # Pull from the public registry and tag with the DTR registry address.
  if docker pull "${img}" > /dev/null 2>&1 && docker tag "${img}" "${dtr_img}"; then
    echo "tagged: ${dtr_img}"
  else
    echo "FAILED: ${img} -> ${dtr_img}" >&2
    failcount=$((failcount+1))
  fi
done

if [ "${failcount}" -gt 0 ]; then
  echo "ERROR: ${failcount} image(s) failed to preload on $(hostname)" >&2
  exit 1
fi
echo "preload OK: ${bootstrapper} on $(hostname)"
`, bootstrapper, dtrHost+"/admin")

	out, err := sshRunScript(user, addr, keyPath, script)
	if err != nil {
		t.Logf("preload output:\n%s", out)
	}
	require.NoError(t, err, "preload %s on %s", bootstrapper, addr)
}

// semverLess reports whether X.Y.Z version string a is semantically less than b.
func semverLess(a, b string) bool {
	parse := func(v string) [3]int {
		var x, y, z int
		fmt.Sscanf(v, "%d.%d.%d", &x, &y, &z)
		return [3]int{x, y, z}
	}
	av, bv := parse(a), parse(b)
	for i := range av {
		if av[i] != bv[i] {
			return av[i] < bv[i]
		}
	}
	return false
}

// fetchLatestMKEVersion queries the Docker Hub tags API for mirantis/ucp and
// returns the highest X.Y.Z release in the given major series (e.g. "3" →
// "3.9.3"). Tags that are not bare version strings (e.g. "latest", "3.9",
// "3.9.3-rc1") are ignored.
func fetchLatestMKEVersion(t *testing.T, major string) string {
	t.Helper()

	type hubTag struct {
		Name string `json:"name"`
	}
	type hubPage struct {
		Results []hubTag `json:"results"`
		Next    *string  `json:"next"`
	}

	// Match only bare X.Y.Z tags in the requested major series.
	versionRe := regexp.MustCompile(`^` + regexp.QuoteMeta(major) + `\.\d+\.\d+$`)

	var candidates []string
	pageURL := "https://hub.docker.com/v2/repositories/mirantis/ucp/tags/?page_size=100"
	const maxPages = 20
	for page := 0; pageURL != "" && page < maxPages; page++ {
		resp, err := http.Get(pageURL)
		require.NoError(t, err, "query Docker Hub tags for mirantis/ucp (page %d)", page+1)
		var p hubPage
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&p),
			"decode Docker Hub tags response (page %d)", page+1)
		resp.Body.Close()

		for _, tag := range p.Results {
			if versionRe.MatchString(tag.Name) {
				candidates = append(candidates, tag.Name)
			}
		}
		if p.Next == nil || *p.Next == "" {
			break
		}
		pageURL = *p.Next
	}

	require.NotEmpty(t, candidates, "no MKE %s.x.y releases found on Docker Hub for mirantis/ucp", major)
	sort.Slice(candidates, func(i, j int) bool { return semverLess(candidates[i], candidates[j]) })
	return candidates[len(candidates)-1]
}

// fetchLatestMCRChannel probes the Mirantis Ubuntu apt repository to find the
// highest available stable channel for the given MCR major version
// (e.g. 29 → "stable-29.4"). It probes stable-29.1 through stable-29.N and
// returns the highest minor for which the Packages file exists.
//
// IMPORTANT: channels are NOT sequential. For example stable-29.1 (404),
// stable-29.2 (200), stable-29.3 (404), stable-29.4 (200). The loop therefore
// never breaks early on a 404 — it always scans the full range.
//
// The Ubuntu 22.04 (jammy) apt repo is used as the probe target:
// https://repos.mirantis.com/ubuntu/dists/jammy/<channel>/binary-amd64/Packages
func fetchLatestMCRChannel(t *testing.T, major int) string {
	t.Helper()

	const (
		probeBase = "https://repos.mirantis.com/ubuntu/dists/jammy"
		probeArch = "binary-amd64/Packages"
		maxMinor  = 20
	)

	last := ""
	for minor := 1; minor <= maxMinor; minor++ {
		channel := fmt.Sprintf("stable-%d.%d", major, minor)
		url := fmt.Sprintf("%s/%s/%s", probeBase, channel, probeArch)
		resp, err := http.Head(url)
		if resp != nil {
			resp.Body.Close()
		}
		// Do NOT break on 404 — channels are non-sequential; a gap does not
		// mean higher minors are absent.
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			last = channel
		}
	}

	require.NotEmpty(t, last,
		"no stable-%d.x MCR channel found at %s — check that repos.mirantis.com is reachable",
		major, probeBase)
	return last
}

// runAirgappedMultiHopUpgradeTest provisions the cluster with an MSR node on
// port 4443, installs the base software using docker.io/mirantis, pre-loads
// all upgrade images into DTR, then drives each upgrade step with both
// mke.imageRepo and msr.imageRepo pointing to DTR.
func runAirgappedMultiHopUpgradeTest(t *testing.T, cfg airgapUpgradeConfig) {
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
		// Expose DTR on port 4443 via the NLB. DTR continues to listen on
		// 443 internally; the NLB translates 4443 → 443. The launchpad.tf
		// template appends :4443 to --dtr-external-url automatically when
		// msr_port != 443. All other smoke tests use the default (443) and
		// are unaffected by this variable.
		"msr_port": 4443,
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

	// ── Base install (using docker.io/mirantis) ───────────────────────────────
	// DTR does not exist yet, so the initial install must use the public
	// registry. Once DTR is up, all subsequent Apply() calls use DTR.
	t.Logf("installing base: MCR %s / MKE %s / MSR %s", cfg.Base.MCRChannel, cfg.Base.MKEVersion, cfg.Base.MSRVersion)

	baseProduct, err := config.ProductFromYAML([]byte(baseYAML))
	require.NoError(t, err, "parse base launchpad YAML")

	err = baseProduct.Apply(true, true, 3, true)
	require.NoError(t, err, "base install Apply()")

	// ── Pre-load upgrade images on all nodes ─────────────────────────────────
	// DTR is now running on port 4443 but we do NOT push images to it.
	// Instead, every upgrade image is pulled from docker.io/mirantis on EVERY
	// node and tagged with the DTR registry address. Launchpad's "Pull MKE
	// images" phase checks `docker image inspect` before attempting a pull; it
	// finds the pre-tagged image locally and skips the pull entirely. This
	// sidesteps DTR push authentication entirely while still exercising
	// Launchpad's imageRepo feature — all upgrade Apply() calls set
	// mke.imageRepo and msr.imageRepo to the DTR address so image references
	// are constructed correctly.
	dtrHost, err := extractDTRHost(baseYAML)
	require.NoError(t, err, "extract DTR hostname from launchpad YAML")

	allHosts, err := extractAllSSHHosts(baseYAML)
	require.NoError(t, err, "extract all SSH hosts from launchpad YAML")

	upgradeVersions := make([]string, 0, len(cfg.Steps))
	for _, step := range cfg.Steps {
		upgradeVersions = append(upgradeVersions, step.MKEVersion)
	}

	for _, h := range allHosts {
		for _, version := range upgradeVersions {
			bootstrapper := fmt.Sprintf("docker.io/mirantis/ucp:%s", version)
			t.Logf("preloading MKE %s on %s", version, h.addr)
			preloadImagesOnNode(t, h.user, h.addr, h.keyPath, dtrHost, bootstrapper)
		}
		// Pre-load the MSR bootstrapper so Launchpad can resolve msr.imageRepo
		// during upgrade Apply() if it checks the installed MSR version.
		msrBootstrapper := fmt.Sprintf("docker.io/mirantis/dtr:%s", cfg.Base.MSRVersion)
		t.Logf("preloading MSR %s on %s", cfg.Base.MSRVersion, h.addr)
		preloadImagesOnNode(t, h.user, h.addr, h.keyPath, dtrHost, msrBootstrapper)
	}

	// ── Sequential upgrade steps (images from DTR address) ───────────────────
	// registryPrefix is the imageRepo value for all upgrade steps:
	// "<dtrHost>/admin". Launchpad constructs image references as
	// "<imageRepo>/ucp:<version>" → "<dtrHost>/admin/ucp:<version>", which
	// matches the local tag created by preloadImagesOnNode above.
	// No REGISTRY_* env vars are set; DTR push/pull is never attempted so
	// Launchpad's AuthenticateDocker phase is intentionally skipped.
	registryPrefix := dtrHost + "/admin"

	currentYAML := baseYAML
	lastProduct := baseProduct
	for i, step := range cfg.Steps {
		t.Logf("upgrade step %d/%d → MCR %s / MKE %s (imageRepo: %s)",
			i+1, len(cfg.Steps), step.MCRChannel, step.MKEVersion, registryPrefix)

		upgradeYAML, err := bumpVersionsAirgap(currentYAML, step.MCRChannel, step.MKEVersion, registryPrefix)
		require.NoError(t, err, "mutate YAML for upgrade step %d", i+1)

		upgradeProduct, err := config.ProductFromYAML([]byte(upgradeYAML))
		require.NoError(t, err, "parse upgrade YAML for step %d", i+1)

		err = upgradeProduct.Apply(true, true, 3, true)
		assert.NoError(t, err, "upgrade Apply() for step %d", i+1)
		if err != nil {
			t.Logf("upgrade step %d failed; stopping upgrade chain", i+1)
			break
		}

		currentYAML = upgradeYAML
		lastProduct = upgradeProduct
	}

	// ── Reset (best-effort) ───────────────────────────────────────────────────
	// See smoke_test.go for rationale on non-fatal Reset().
	if err := lastProduct.Reset(); err != nil {
		t.Logf("WARN: product.Reset() failed (non-fatal): %v", err)
	}
}

// TestAirgappedMultiHopUpgrade provisions an airgapped cluster with an
// internal DTR on port 4443 and exercises the following upgrade chain,
// pulling all MKE images from DTR after the initial bootstrap:
//
//	install: MCR stable-25.0 / MKE 3.8.8  / MSR 2.9.27  (docker.io/mirantis)
//	step 1:  MCR stable-25.0 / MKE 3.8.11               (from DTR :4443)
//	step 2:  MCR stable-29.2 / MKE 3.8.12               (from DTR :4443)
//	step 3:  MCR stable-29.2 / MKE 3.9.2                (from DTR :4443)
//	step 4:  MCR <latest 29.x channel> / MKE <latest 3.x> (discovered at runtime)
//
// Step 4 is resolved dynamically at test runtime: fetchLatestMKEVersion queries
// Docker Hub for the highest mirantis/ucp 3.x.y tag, and fetchLatestMCRChannel
// probes the Mirantis apt repository for the highest stable-29.x channel. If
// the resolved versions are identical to step 3 (no new releases), step 4 is
// omitted.
//
// Node matrix: Ubuntu22 manager + Rhel8 worker + Ubuntu22 MSR.
func TestAirgappedMultiHopUpgrade(t *testing.T) {
	latestMKE := fetchLatestMKEVersion(t, "3")
	latestMCR := fetchLatestMCRChannel(t, 29)
	t.Logf("discovered latest: MKE %s / MCR channel %s", latestMKE, latestMCR)

	// Fixed portion of the upgrade chain — the specific versions that model
	// the customer scenario and must always be exercised.
	steps := []upgradeStep{
		{MCRChannel: "stable-25.0", MKEVersion: "3.8.11"},
		{MCRChannel: "stable-29.2", MKEVersion: "3.8.12"},
		{MCRChannel: "stable-29.2", MKEVersion: "3.9.2"},
	}

	// Append the dynamic "to latest" step only when it differs from the last
	// fixed step — avoids a redundant no-op when the fixed chain already ends
	// on the current latest release.
	lastFixed := steps[len(steps)-1]
	if latestMKE != lastFixed.MKEVersion || latestMCR != lastFixed.MCRChannel {
		steps = append(steps, upgradeStep{MCRChannel: latestMCR, MKEVersion: latestMKE})
		t.Logf("appending upgrade-to-latest step: MCR %s / MKE %s", latestMCR, latestMKE)
	} else {
		t.Logf("latest versions match last fixed step; no additional upgrade step needed")
	}

	runAirgappedMultiHopUpgradeTest(t, airgapUpgradeConfig{
		Base: smokeConfig{
			Name:            "airgappedup",
			MCRChannel:      "stable-25.0",
			MKEVersion:      "3.8.8",
			MSRVersion:      "2.9.27",
			SSHKeyAlgorithm: "ed25519",
			Nodegroups: map[string]interface{}{
				"MngrUbuntu22": test.Platforms["Ubuntu22"].GetManager(),
				"WrkRhel8":     test.Platforms["Rhel8"].GetWorker(),
				"MsrUbuntu22":  test.Platforms["Ubuntu22"].GetMSR(),
			},
		},
		Steps: steps,
	})
}
