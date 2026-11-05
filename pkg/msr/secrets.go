package msr

import (
	"fmt"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/Mirantis/launchpad/pkg/util/secretenv"
)

// secretEnvByFlag maps the bootstrapper flags that carry a credential to the
// environment variable it also reads them from, as documented in the MSR
// install CLI reference for 2.8 and 2.9.
//
// The TLS values are PEM blocks: they are staged as environment values
// rather than written into the flag list, so neither a command line nor a
// docker --env-file, which cannot hold a multi-line value, has to carry them.
var secretEnvByFlag = map[string]string{
	"--dtr-ca":              "MSR_CA",
	"--dtr-cert":            "MSR_CERT",
	"--dtr-key":             "MSR_KEY",
	"--ucp-ca":              "UCP_CA",
	"--ucp-username":        "UCP_USERNAME",
	"--ucp-password":        "UCP_PASSWORD",
	"--registry-username":   "REGISTRY_USERNAME",
	"--registry-password":   "REGISTRY_PASSWORD",
	"--client-cert-auth-ca": "CLIENT_CA",
}

// StageSecrets moves every credential in flags, plus the values in extra, out
// of the command line and into mode-0600 files on the host. It returns the
// staged variables and the flags that remain.
//
// Every MSR bootstrapper invocation goes through this: install, join, remove
// and destroy all pass the MKE administrator password, and the install also
// passes the MSR TLS private key. The caller must Remove the returned
// variables once the command has finished, on every path.
func StageSecrets(host secretenv.Host, flags commonconfig.Flags, extra map[string]string) (*secretenv.Vars, commonconfig.Flags, error) {
	values, kept := secretenv.Extract(flags, secretEnvByFlag)

	// A value taken from the flags wins over one the caller supplied: the
	// flags are what the user wrote in spec.msr.installFlags, and the
	// phases previously left an explicit flag in place.
	for name, value := range extra {
		if _, fromFlag := values[name]; !fromFlag && value != "" {
			values[name] = value
		}
	}

	staged, err := secretenv.Stage(host, values)
	if err != nil {
		return nil, flags, fmt.Errorf("failed to stage MSR credentials: %w", err)
	}

	return staged, commonconfig.Flags(kept), nil
}
