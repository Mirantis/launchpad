package configurer

import (
	"fmt"
	"io"
	"io/fs"
	"testing"

	"github.com/k0sproject/rig/exec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHost stubs os.Host for unit-testing configurer methods that run remote
// commands. Command responses are keyed by the full command string; a missing
// key returns an error to simulate command-not-found / non-zero exit.
type mockHost struct {
	outputs map[string]string
	errors  map[string]error
}

func (m *mockHost) String() string { return "mockHost" }

func (m *mockHost) ExecOutput(cmd string, _ ...exec.Option) (string, error) {
	if err, ok := m.errors[cmd]; ok {
		return "", err
	}
	if out, ok := m.outputs[cmd]; ok {
		return out, nil
	}
	return "", fmt.Errorf("unexpected command: %q", cmd)
}

func (m *mockHost) Exec(cmd string, opts ...exec.Option) error {
	_, err := m.ExecOutput(cmd, opts...)
	return err
}

func (m *mockHost) ExecOutputf(cmd string, argsOrOpts ...any) (string, error) {
	// Separate format args from exec.Option values.
	var args []any
	var opts []exec.Option
	for _, a := range argsOrOpts {
		if o, ok := a.(exec.Option); ok {
			opts = append(opts, o)
		} else {
			args = append(args, a)
		}
	}
	return m.ExecOutput(fmt.Sprintf(cmd, args...), opts...)
}

func (m *mockHost) Execf(cmd string, argsOrOpts ...any) error {
	_, err := m.ExecOutputf(cmd, argsOrOpts...)
	return err
}

func (m *mockHost) Upload(_ string, _ string, _ fs.FileMode, _ ...exec.Option) error {
	return nil
}

func (m *mockHost) ExecStreams(_ string, _ io.ReadCloser, _ io.Writer, _ io.Writer, _ ...exec.Option) (exec.Waiter, error) {
	return nil, nil
}

func (m *mockHost) Sudo(cmd string) (string, error) {
	return cmd, nil
}

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
