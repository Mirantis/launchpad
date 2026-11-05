package test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

// Environment variables the examples' generated launchpad.yaml refers to
// instead of inlining credentials. launchpad expands them through envsubst
// when it loads the config, so a consumer of that YAML must export them
// before parsing it or config loading fails on an unset reference.
const (
	MKEAdminPasswordEnv = "LAUNCHPAD_MKE_ADMIN_PASSWORD"
	WinRMPasswordEnv    = "LAUNCHPAD_WINRM_PASSWORD"
)

// AdminCIDRs returns the value for the examples' admin_cidrs variable, which
// scopes SSH, WinRM and the MKE/Kubernetes API to named networks.
//
// The variable is required and has no default, so that narrowing it is a
// decision a caller makes rather than something a module silently decides.
// These suites provision throwaway stacks from runners whose egress address
// is not known ahead of time, so the fallback is 0.0.0.0/0: the same exposure
// the examples had before, but stated at the caller where it can be found and
// changed. Set SMOKE_ADMIN_CIDRS (comma separated) to narrow it.
func AdminCIDRs(t *testing.T) []string {
	t.Helper()

	configured := os.Getenv("SMOKE_ADMIN_CIDRS")
	if configured == "" {
		t.Logf("SMOKE_ADMIN_CIDRS is unset: provisioning with admin_cidrs = [0.0.0.0/0]")

		return []string{"0.0.0.0/0"}
	}

	cidrs := strings.Split(configured, ",")
	for i, cidr := range cidrs {
		cidrs[i] = strings.TrimSpace(cidr)
	}

	return cidrs
}

// ExportGeneratedCredentials publishes the credentials the generated
// launchpad.yaml refers to, so that envsubst resolves them when the config is
// loaded. Pass an empty windowsPassword for a stack with no Windows nodes;
// the reference is only present in the document when a WinRM host is.
func ExportGeneratedCredentials(t *testing.T, mkePassword, windowsPassword string) {
	t.Helper()

	t.Setenv(MKEAdminPasswordEnv, mkePassword)
	t.Setenv(WinRMPasswordEnv, windowsPassword)
}

// TerraformOptions builds terraform.Options that keep credentials out of the
// terraform command line.
//
// vars are rendered as "-var key=value" arguments, so they end up in the
// process argv: terratest logs that argv verbatim (terratest
// modules/shell/command.go logs "Running command %s with args %s"), and any
// process on the same machine can read it from /proc/<pid>/cmdline. No logger
// setting can fix argv, so credentials must not travel that way.
//
// secretVars are passed as TF_VAR_<name> entries in Options.EnvVars instead.
// terratest puts EnvVars on the child process environment and never logs them.
// Values that are not strings are JSON encoded, which terraform parses as the
// HCL expression for the variable's complex type.
//
// Because a non-string value is parsed as an HCL expression rather than taken
// literally, a credential inside one must not contain "${" or "%{": terraform
// reads those as template interpolation sequences and fails with "Invalid
// expression" or "Unexpected end of template" before the variable is ever
// used. No generated credential contains them today -- the MKE password is
// alphanumeric and the Windows charset excludes "$" -- but smoke_test.go
// carries a TODO to restore "$" to that charset (PRODENG-3751), which would
// put the first half of "${" back in play. A string secret is passed verbatim
// and is not affected.
func TerraformOptions(t *testing.T, terraformDir string, vars map[string]any, secretVars map[string]any) terraform.Options {
	t.Helper()

	envVars := make(map[string]string, len(secretVars))

	for name, value := range secretVars {
		if str, ok := value.(string); ok {
			envVars["TF_VAR_"+name] = str

			continue
		}

		// json.Marshal cannot fail for the values these suites pass, which
		// are maps of strings, bools and numbers; the branch exists so an
		// unmarshalable value is reported rather than silently dropping the
		// variable.
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("TerraformOptions: encoding secret variable %q failed: %v", name, err)
		}

		envVars["TF_VAR_"+name] = string(encoded)
	}

	return terraform.Options{
		TerraformDir: terraformDir,
		Vars:         vars,
		EnvVars:      envVars,
	}
}

// OutputSecret returns a terraform output whose value is a credential.
//
// terraform output streams the value on stdout, and terratest copies every
// stdout line to the logger, so for these outputs the logged text is the
// secret itself. The logger is discarded for this one command rather than for
// the whole Options, so apply and destroy keep logging: their output is the
// only artifact a failed provisioning run leaves behind, and it carries no
// credential once the secrets travel by TF_VAR_*.
//
// Discarding rather than redacting is deliberate. Redaction can only mask
// strings the caller knows, and these outputs also carry credentials the
// terraform module generated, which the caller never sees.
func OutputSecret(t *testing.T, options *terraform.Options, name string) string {
	t.Helper()

	return terraform.Output(t, withDiscardedLogger(options), name)
}

// OutputSecretJSON returns a terraform output whose value is a credential, as
// JSON. See OutputSecret for why the logger is discarded.
func OutputSecretJSON(t *testing.T, options *terraform.Options, name string) string {
	t.Helper()

	return terraform.OutputJson(t, withDiscardedLogger(options), name)
}

// withDiscardedLogger returns a copy of options that logs nothing. The copy is
// shallow on purpose: Vars, EnvVars and the retryable-error maps stay shared
// with the caller's Options, while Logger is replaced only on the copy, so the
// caller's own Options keeps logging.
func withDiscardedLogger(options *terraform.Options) *terraform.Options {
	quiet := *options
	quiet.Logger = logger.Discard

	return &quiet
}
