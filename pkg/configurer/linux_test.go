package configurer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/cmd"
	"github.com/k0sproject/rig/v2/protocol"
	"github.com/k0sproject/rig/v2/remotefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHost stubs configurer.Host for unit-testing configurer methods that run
// remote commands. Command responses are keyed by the full command string; a
// missing key returns an error to simulate command-not-found / non-zero exit.
// Only ExecOutput is exercised by LocalAddresses; the rest of the Host
// interface is stubbed to satisfy the type.
type mockHost struct {
	outputs map[string]string
	errors  map[string]error
}

func (m *mockHost) String() string  { return "mockHost" }
func (m *mockHost) IsWindows() bool { return false }

func (m *mockHost) ExecOutput(cmdStr string, _ ...cmd.ExecOption) (string, error) {
	if err, ok := m.errors[cmdStr]; ok {
		return "", err
	}
	if out, ok := m.outputs[cmdStr]; ok {
		return out, nil
	}
	return "", fmt.Errorf("unexpected command: %q", cmdStr)
}

func (m *mockHost) Exec(cmdStr string, opts ...cmd.ExecOption) error {
	_, err := m.ExecOutput(cmdStr, opts...)
	return err
}

func (m *mockHost) ExecReader(cmdStr string, _ ...cmd.ExecOption) io.Reader {
	out, err := m.ExecOutput(cmdStr)
	if err != nil {
		return &errReader{err: err}
	}
	return strings.NewReader(out)
}

func (m *mockHost) ExecScanner(cmdStr string, opts ...cmd.ExecOption) *bufio.Scanner {
	return bufio.NewScanner(m.ExecReader(cmdStr, opts...))
}

func (m *mockHost) StartBackground(_ string, _ ...cmd.ExecOption) (protocol.Waiter, error) {
	return nil, fmt.Errorf("not implemented in mockHost")
}

func (m *mockHost) ExecContext(_ context.Context, cmdStr string, opts ...cmd.ExecOption) error {
	return m.Exec(cmdStr, opts...)
}

func (m *mockHost) ExecOutputContext(_ context.Context, cmdStr string, opts ...cmd.ExecOption) (string, error) {
	return m.ExecOutput(cmdStr, opts...)
}

func (m *mockHost) ExecReaderContext(_ context.Context, cmdStr string, opts ...cmd.ExecOption) io.Reader {
	return m.ExecReader(cmdStr, opts...)
}

func (m *mockHost) Start(_ context.Context, cmdStr string, opts ...cmd.ExecOption) (protocol.Waiter, error) {
	return m.StartBackground(cmdStr, opts...)
}

func (m *mockHost) Sudo() *rig.Client { return nil }

func (m *mockHost) FS() remotefs.FS { return nil }

type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }

// TestLocalAddresses_Hostname verifies the happy path: hostname --all-ip-addresses
// returns a space-separated list (with trailing space, as real hostname emits).
func TestLocalAddresses_Hostname(t *testing.T) {
	h := &mockHost{
		outputs: map[string]string{
			"hostname --all-ip-addresses": "10.0.0.5 172.31.0.10 ",
		},
	}
	addrs, err := LinuxConfigurer{}.LocalAddresses(h)
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.5", "172.31.0.10"}, addrs)
}

// TestLocalAddresses_IPFallback verifies the SLES 12 SP5 scenario: hostname
// --all-ip-addresses fails (exit 4), so LocalAddresses falls back to
// "ip -4 -o addr show scope global" and parses the addr/prefix fields.
func TestLocalAddresses_IPFallback(t *testing.T) {
	ipOutput := "2: eth0    inet 10.0.0.5/24 brd 10.0.0.255 scope global eth0\\\n" +
		"3: eth1    inet 172.31.0.10/20 brd 172.31.15.255 scope global eth1\\\n"
	h := &mockHost{
		outputs: map[string]string{
			"ip -4 -o addr show scope global": ipOutput,
		},
		errors: map[string]error{
			"hostname --all-ip-addresses": fmt.Errorf("exit status 4"),
		},
	}
	addrs, err := LinuxConfigurer{}.LocalAddresses(h)
	require.NoError(t, err)
	assert.Equal(t, []string{"10.0.0.5", "172.31.0.10"}, addrs)
}

// TestLocalAddresses_BothFail verifies that an error is returned when both
// commands are unavailable (neither hostname nor ip work).
func TestLocalAddresses_BothFail(t *testing.T) {
	h := &mockHost{
		errors: map[string]error{
			"hostname --all-ip-addresses":     fmt.Errorf("exit status 4"),
			"ip -4 -o addr show scope global": fmt.Errorf("exit status 127"),
		},
	}
	_, err := LinuxConfigurer{}.LocalAddresses(h)
	assert.Error(t, err)
}
