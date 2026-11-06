package mke

import (
	"io"
	"os"
	"strings"
	"testing"

	common "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/Mirantis/launchpad/pkg/product/mke/config"
	rig "github.com/k0sproject/rig/v2"
	rigcmd "github.com/k0sproject/rig/v2/cmd"
	"github.com/k0sproject/rig/v2/protocol/winrm"
	"github.com/stretchr/testify/require"
)

// Credentials reach "describe config" from three places: the MKE admin
// password field, a host connection password owned by rig, and a secret left
// inside a free-form install flag. None of them may be printed.
func TestDescribeConfigRedactsCredentials(t *testing.T) {
	product := productWithCredentials()

	out := captureStdout(t, func() {
		require.NoError(t, product.Describe("config"))
	})

	for _, secret := range []string{
		"fake-admin-password",
		"fake-host-password",
		"fake-flag-password",
		"fake-admin-user",
	} {
		require.NotContainsf(t, out, secret, "describe config leaked %q:\n%s", secret, out)
	}

	// The report is still useful: non-credential configuration survives.
	require.Contains(t, out, "test-cluster")
	require.Contains(t, out, "10.0.0.1")
	require.Contains(t, out, "--san=mke.example.com")
}

// --disable-redact is launchpad's documented opt-out for hiding sensitive
// information in its output.
func TestDescribeConfigHonoursDisableRedact(t *testing.T) {
	rigcmd.DisableRedact = true
	t.Cleanup(func() { rigcmd.DisableRedact = false })

	out := captureStdout(t, func() {
		require.NoError(t, productWithCredentials().Describe("config"))
	})

	require.Contains(t, out, "fake-admin-password")
	require.Contains(t, out, "fake-host-password")
}

func productWithCredentials() *MKE {
	return &MKE{ClusterConfig: config.ClusterConfig{
		APIVersion: "launchpad.mirantis.com/mke/v1.6",
		Kind:       "mke",
		Metadata:   &config.ClusterMeta{Name: "test-cluster"},
		Spec: &config.ClusterSpec{
			Hosts: config.Hosts{{
				CompositeConfig: rig.CompositeConfig{
					WinRM: &winrm.Config{
						Address:  "10.0.0.1",
						User:     "Administrator",
						Password: "fake-host-password",
					},
				},
				Role: "manager",
			}},
			MKE: config.MKEConfig{
				Version:       "3.7.0",
				AdminUsername: "fake-admin-user",
				AdminPassword: "fake-admin-password",
				InstallFlags:  common.Flags{"--san=mke.example.com", "--ucp-password=fake-flag-password"},
			},
		},
	}}
}

// Output larger than the 64 KiB pipe buffer must not wedge the report or the
// harness capturing it. caCertData is unmasked by design, so a realistic CA
// bundle is exactly how a config gets there.
func TestDescribeConfigHandlesLargeOutput(t *testing.T) {
	product := productWithCredentials()
	product.ClusterConfig.Spec.MKE.CACertData = strings.Repeat("ca-bundle-line\n", 10000)

	out := captureStdout(t, func() {
		require.NoError(t, product.Describe("config"))
	})

	require.Greater(t, len(out), 64*1024)
	require.NotContains(t, out, "fake-admin-password")
}

// captureStdout drains the pipe while fn writes to it. Reading only after fn
// returns deadlocks once the output exceeds the 64 KiB pipe buffer, which a
// config carrying realistic certificate or configData blobs does.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	read, write, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = write

	defer func() {
		os.Stdout = orig
	}()

	captured := make(chan []byte, 1)

	go func() {
		out, _ := io.ReadAll(read)
		captured <- out
	}()

	fn()
	require.NoError(t, write.Close())

	return string(<-captured)
}
