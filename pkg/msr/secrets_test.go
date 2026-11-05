package msr_test

import (
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/msr"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/k0sproject/rig/v2/remotefs"
	"github.com/stretchr/testify/require"
)

// stagingFS stands in for a managed host's filesystem.
type stagingFS struct {
	remotefs.FS

	written map[string]string
	count   int
}

func (f *stagingFS) CreateTemp(_, prefix string) (string, error) {
	f.count++

	return fmt.Sprintf("/tmp/%s%d", prefix, f.count), nil
}

func (f *stagingFS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	f.written[path] = string(data)

	return nil
}

func (f *stagingFS) Remove(path string) error {
	delete(f.written, path)

	return nil
}

type stagingHost struct{ fs *stagingFS }

func (h stagingHost) FS() remotefs.FS { return h.fs }

// Every MSR bootstrapper invocation carries the MKE administrator password,
// and the install also carries the MSR TLS private key. None of them may
// reach the command line, where /proc/<pid>/cmdline and auditd execve records
// would keep them.
func TestStageSecretsKeepsCredentialsOutOfTheCommand(t *testing.T) {
	host := stagingHost{fs: &stagingFS{written: map[string]string{}}}

	flags := commonconfig.Flags{
		"--ucp-url=\"mke.example.com\"",
		"--ucp-username=\"fake-admin-user\"",
		"--ucp-password=\"fake-admin-password\"",
		"--ucp-node host-1",
	}

	staged, kept, err := msr.StageSecrets(host, flags, map[string]string{
		"MSR_KEY": "fake-key-line-one\nfake-key-line-two\n",
		"MSR_CA":  "",
	})
	require.NoError(t, err)

	command := staged.Assignments() + " docker run " + strings.Join(staged.PassFlags(), " ") + " mirantis/dtr install " + kept.Join()

	for _, secret := range []string{"fake-admin-password", "fake-admin-user", "fake-key-line-one"} {
		require.NotContainsf(t, command, secret, "%q reached the remote command line:\n%s", secret, command)
	}

	// Non-credential flags are untouched, so the installer is still asked
	// to do the same thing.
	require.Contains(t, command, `--ucp-url="mke.example.com"`)
	require.Contains(t, command, "--ucp-node host-1")
	require.Contains(t, command, "-e UCP_PASSWORD")
	require.Contains(t, command, "-e MSR_KEY")

	// An empty value is not staged: there is no secret and no file to leak.
	require.NotContains(t, command, "-e MSR_CA")

	require.Equal(t, []string{"MSR_KEY", "UCP_PASSWORD", "UCP_USERNAME"}, staged.Names())
	require.Len(t, host.fs.written, 3)

	require.NoError(t, staged.Remove())
	require.Empty(t, host.fs.written, "a staged credential outlived the command")
}
