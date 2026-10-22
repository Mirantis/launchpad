package msr

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/k0sproject/rig/v2/sh/shellescape"
	"github.com/stretchr/testify/require"
)

func TestPluckSharedInstallFlags(t *testing.T) {

	t.Run("Install flags are shared with join", func(t *testing.T) {
		installFlags := []string{
			"--replica-http-port 8000",
			"--replica-https-port 4443",
			"--ucp-insecure-tls",
			"--nfs-storage-url nfs://nfs.acme.com",
		}
		expectedJoinFlags := []string{
			"--replica-http-port 8000",
			"--replica-https-port 4443",
			"--ucp-insecure-tls",
		}
		actualJoinFlags := PluckSharedInstallFlags(installFlags, SharedInstallJoinFlags)
		sort.Strings(actualJoinFlags)
		sort.Strings(expectedJoinFlags)
		if !reflect.DeepEqual(actualJoinFlags, expectedJoinFlags) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expectedJoinFlags, actualJoinFlags)
		}
	})

	t.Run("Flags with multiple values can still be plucked", func(t *testing.T) {
		multiValueFlag := []string{
			"--fake-flag one two three",
		}
		testSharedFlags := []string{
			"--fake-flag",
		}
		expected := []string{
			"--fake-flag one two three",
		}
		actual := PluckSharedInstallFlags(multiValueFlag, testSharedFlags)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expected, actual)
		}
	})
}

func TestBuildMKEFlags(t *testing.T) {
	config := &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{
				AdminUsername: "admin",
				AdminPassword: "password1234",
				InstallFlags: []string{
					"--san ucp.acme.com",
				},
			},
			MSR: &mkeconfig.MSRConfig{},
		},
	}

	t.Run("MKE flags are built shell-quoted when --san is provided", func(t *testing.T) {
		actual := BuildMKEFlags(config)
		expected := commonconfig.Flags{
			"--ucp-url ucp.acme.com",
			"--ucp-username admin",
			"--ucp-password password1234",
		}
		sort.Strings(actual)
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expected, actual)
		}
	})
}

// TestBuildMKEFlagsQuotesCredentials pins the property every MSR bootstrapper
// phase depends on: the flag values are single shell-safe words, so a
// password containing $(...), backticks or $VAR cannot execute when the flag
// list is joined and run through the remote shell.
func TestBuildMKEFlagsQuotesCredentials(t *testing.T) {
	t.Parallel()

	config := &mkeconfig.ClusterConfig{Spec: &mkeconfig.ClusterSpec{}}
	config.Spec.MKE = mkeconfig.MKEConfig{
		AdminUsername: `ad'min"$(touch /tmp/pwned)`,
		AdminPassword: "p$(touch /tmp/pwned)`id`$HOME",
	}
	config.Spec.MSR = &mkeconfig.MSRConfig{
		InstallFlags: commonconfig.Flags{"--ucp-url 10.0.0.1"},
	}
	flags := BuildMKEFlags(config)

	for _, flag := range []string{"--ucp-username", "--ucp-password"} {
		value := flags.GetValue(flag)
		require.Equal(t, value, strings.TrimSpace(value), "%s value is a single word: %q", flag, value)
	}

	require.Equal(t,
		`ad'min"$(touch /tmp/pwned)`,
		mustUnquote(t, flags.GetValue("--ucp-username")),
	)
	require.Equal(t,
		"p$(touch /tmp/pwned)`id`$HOME",
		mustUnquote(t, flags.GetValue("--ucp-password")),
	)
}

// TestDestroyQuotesCredentials pins the single-word property the remote
// shell depends on: DockerCmd quoting of a metacharacter payload must
// survive shellescape.Unquote unchanged and stay wrapped, so the shell sees
// one literal argument instead of expanding it.
func TestDestroyQuotesCredentials(t *testing.T) {
	t.Parallel()

	var c configurer.LinuxConfigurer
	const pass = "p`id`$HOME"

	quotedPass := c.DockerCmd().Quote(pass)

	require.Contains(t, quotedPass, `'`, "quoting must wrap the value")
	require.Equal(t, pass, mustUnquote(t, quotedPass), "quoting must be reversible without payload loss")
}

func mustUnquote(t *testing.T, s string) string {
	t.Helper()

	unquoted, err := shellescape.Unquote(s)
	require.NoError(t, err)

	return unquoted
}

func TestFormatReplicaID(t *testing.T) {
	require.Equal(t, "000000000001", FormatReplicaID(1))
	require.Equal(t, "00000000000a", FormatReplicaID(10))
	require.Equal(t, "000000000010", FormatReplicaID(16))
}

func TestSequentialReplicaIDs(t *testing.T) {
	config := &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: []*mkeconfig.Host{
				{Role: "msr"},
				{Role: "msr", MSRMetadata: &mkeconfig.MSRMetadata{ReplicaID: "00000000001f"}},
				{Role: "msr"},
			},
			MSR: &mkeconfig.MSRConfig{ReplicaIDs: "sequential"},
		},
	}
	require.NoError(t, AssignSequentialReplicaIDs(config))
	require.Equal(t, "000000000020", config.Spec.Hosts[0].MSRMetadata.ReplicaID)
	require.Equal(t, "00000000001f", config.Spec.Hosts[1].MSRMetadata.ReplicaID)
	require.Equal(t, "000000000021", config.Spec.Hosts[2].MSRMetadata.ReplicaID)
}
