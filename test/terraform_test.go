package test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mkePassword     = "mke-admin-password-fixture"
	windowsPassword = "windows-administrator-password-fixture"
)

// captureLogger records everything terratest logs, so a test can assert on
// what a real run would have written to the CI log.
type captureLogger struct {
	mu    sync.Mutex
	lines []string
}

func (c *captureLogger) Logf(_ terratesting.TestingT, format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lines = append(c.lines, fmt.Sprintf(format, args...))
}

func (c *captureLogger) String() string {
	return c.since(0)
}

// lineCount returns how many lines have been captured so far, so a caller can
// assert on what a later command logged rather than on the whole log.
func (c *captureLogger) lineCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.lines)
}

// since returns the lines captured after the first n lines.
func (c *captureLogger) since(n int) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return strings.Join(c.lines[n:], "\n")
}

// fixtureVars returns the variable split the smoke and integration suites use:
// everything that is not a credential as a "-var" argument, the credentials as
// TF_VAR_* environment entries.
func fixtureVars(name string) (vars map[string]any, secretVars map[string]any) {
	return map[string]any{
			"name": name,
		},
		map[string]any{
			"launchpad": map[string]any{
				"drain":       false,
				"mcr_channel": "stable-25.0",
				"mke_version": "3.8.8",
				"msr_version": "2.9.28",
				"mke_connect": map[string]any{
					"username": "admin",
					"password": mkePassword,
					"insecure": true,
				},
			},
			"windows_password": windowsPassword,
		}
}

func TestTerraformOptionsKeepsCredentialsOutOfTheCommandLine(t *testing.T) {
	vars, secretVars := fixtureVars("argv-check")

	options := TerraformOptions(t, "../examples/terraform/aws-simple", vars, secretVars)

	// FormatTerraformVarsAsArgs renders exactly the "-var" arguments terraform
	// is executed with, which is both what terratest logs and what any process
	// on the machine can read from /proc/<pid>/cmdline.
	args := strings.Join(terraform.FormatTerraformVarsAsArgs(options.Vars), " ")
	assert.NotContains(t, args, mkePassword, "MKE admin password must not reach the terraform argv")
	assert.NotContains(t, args, windowsPassword, "Windows password must not reach the terraform argv")
	assert.Contains(t, args, "argv-check", "non-credential variables must still travel as -var")

	// The credentials travel in the environment instead.
	assert.Contains(t, options.EnvVars["TF_VAR_launchpad"], mkePassword)
	assert.Equal(t, windowsPassword, options.EnvVars["TF_VAR_windows_password"],
		"a string secret is passed verbatim, not JSON quoted")

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(options.EnvVars["TF_VAR_launchpad"]), &decoded),
		"a complex secret variable must be encoded as JSON for terraform to parse it")
}

// TestSecretVarsRoundTripThroughTerraform runs a real terraform init, apply and
// output against the provider-less fixture in test/fixtures/tfvars. It proves
// three things that cannot be proven by inspecting the options alone:
//
//  1. TF_VAR_* transports the launchpad object intact, nested password
//     included, so moving it off "-var" has not changed what terraform sees;
//  2. apply's log, which the suites keep for debugging, contains neither
//     password, because the outputs carrying them are marked sensitive;
//  3. OutputSecret fetches a credential-bearing output without logging it,
//     while the surrounding options keep logging normally.
//
// No provider, no resource and no network: the module declares no
// required_providers, so terraform init contacts no registry.
func TestSecretVarsRoundTripThroughTerraform(t *testing.T) {
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform binary not found; skipping the end-to-end variable transport check")
	}

	vars, secretVars := fixtureVars("roundtrip-check")

	options := TerraformOptions(t, copyFixture(t, "fixtures/tfvars"), vars, secretVars)
	captured := &captureLogger{}
	options.Logger = logger.New(captured)

	terraform.InitAndApply(t, &options)

	applyLog := captured.String()
	require.NotEmpty(t, applyLog, "the apply log is the only artifact a failed provisioning run leaves behind")
	assert.Contains(t, applyLog, "Running command terraform with args [apply",
		"apply must keep logging its command line")
	assert.NotContains(t, applyLog, mkePassword, "apply must not log the MKE admin password")
	assert.NotContains(t, applyLog, windowsPassword, "apply must not log the Windows password")

	// The object survived the TF_VAR_* transport: every attribute that was
	// passed comes back with the same value, nested credential included.
	// terraform also fills in mcr_repo_url, which the variable declares as
	// optional with a default, so the echo is a superset rather than equal.
	echoed := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(OutputSecretJSON(t, &options, "launchpad_echo")), &echoed))

	for attribute, sent := range secretVars["launchpad"].(map[string]any) {
		assert.Equal(t, sent, echoed[attribute],
			"terraform must parse TF_VAR_launchpad.%s into exactly what was passed", attribute)
	}

	assert.Equal(t, mkePassword, echoed["mke_connect"].(map[string]any)["password"],
		"the nested credential must survive the TF_VAR_* transport")

	// A credential-bearing output is still readable by the test...
	clusterYAML := OutputSecret(t, &options, "cluster_yaml")
	assert.Contains(t, clusterYAML, mkePassword)
	assert.Contains(t, clusterYAML, windowsPassword)

	// ...without the value reaching the log, while a non-credential output
	// read through the ordinary path still logs its value.
	assert.NotContains(t, captured.String(), mkePassword, "OutputSecret must not log the value it fetched")
	assert.NotContains(t, captured.String(), windowsPassword, "OutputSecret must not log the value it fetched")

	// The log already contains "roundtrip-check" from the apply command line,
	// so only what this call appends can show the caller's logger still works.
	beforeOrdinaryOutput := captured.lineCount()

	assert.Equal(t, "roundtrip-check", terraform.Output(t, &options, "name_echo"))

	ordinaryOutputLog := captured.since(beforeOrdinaryOutput)
	assert.Contains(t, ordinaryOutputLog, "Running command terraform with args [output",
		"discarding the logger for secret outputs must not disable logging for the others")
	assert.Contains(t, ordinaryOutputLog, "roundtrip-check",
		"a non-credential output must still have its value logged")
}

// copyFixture copies a fixture module into a temporary directory, so terraform
// state and the .terraform directory never land in the repository.
func copyFixture(t *testing.T, fixtureDir string) string {
	t.Helper()

	dest := t.TempDir()

	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := os.ReadFile(filepath.Join(fixtureDir, entry.Name())) // #nosec G304 -- fixture path is a test constant
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dest, entry.Name()), content, 0o600))
	}

	return dest
}

func TestAdminCIDRs(t *testing.T) {
	t.Run("falls back to the open range when unset", func(t *testing.T) {
		t.Setenv("SMOKE_ADMIN_CIDRS", "")

		assert.Equal(t, []string{"0.0.0.0/0"}, AdminCIDRs(t))
	})

	t.Run("honours a configured list", func(t *testing.T) {
		t.Setenv("SMOKE_ADMIN_CIDRS", "203.0.113.4/32, 198.51.100.0/24")

		assert.Equal(t, []string{"203.0.113.4/32", "198.51.100.0/24"}, AdminCIDRs(t))
	})
}
