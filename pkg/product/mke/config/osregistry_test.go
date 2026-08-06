package config

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/k0sproject/rig/v2/cmd"
	rigos "github.com/k0sproject/rig/v2/os"
	ps "github.com/k0sproject/rig/v2/powershell"
	"github.com/k0sproject/rig/v2/protocol"
	"github.com/stretchr/testify/require"
)

var errFakeCommandFailed = errors.New("command failed")

// fakeRunner is a cmd.SimpleRunner that answers only the handful of probes the
// OS resolvers issue, driven by a canned command->output table.
type fakeRunner struct {
	windows bool
	out     map[string]string
}

func (f *fakeRunner) String() string  { return "fake" }
func (f *fakeRunner) IsWindows() bool { return f.windows }
func (f *fakeRunner) Exec(cmdStr string, _ ...cmd.ExecOption) error {
	_, err := f.ExecOutput(cmdStr)

	return err
}

func (f *fakeRunner) ExecOutput(cmdStr string, _ ...cmd.ExecOption) (string, error) {
	if v, ok := f.out[cmdStr]; ok {
		return v, nil
	}

	return "", errFakeCommandFailed
}

func (f *fakeRunner) ExecReader(cmdStr string, _ ...cmd.ExecOption) io.Reader {
	out, err := f.ExecOutput(cmdStr)
	if err != nil {
		return &errReader{err: err}
	}

	return strings.NewReader(out)
}

func (f *fakeRunner) ExecScanner(cmdStr string, opts ...cmd.ExecOption) *bufio.Scanner {
	return bufio.NewScanner(f.ExecReader(cmdStr, opts...))
}

func (f *fakeRunner) StartBackground(_ string, _ ...cmd.ExecOption) (protocol.Waiter, error) {
	return nil, errFakeCommandFailed
}

type errReader struct{ err error }

func (e *errReader) Read([]byte) (int, error) { return 0, e.err }

const ubuntuOSRelease = "PRETTY_NAME=\"Ubuntu 22.04.5 LTS\"\nNAME=\"Ubuntu\"\nVERSION_ID=\"22.04\"\nID=ubuntu\nID_LIKE=debian\n"

func linuxRunner() *fakeRunner {
	return &fakeRunner{out: map[string]string{
		"uname | grep -q Linux":                          "",
		"cat /etc/os-release || cat /usr/lib/os-release": ubuntuOSRelease,
		"uname -m":                            "x86_64",
		"command -v apt-get > /dev/null 2>&1": "",
	}}
}

// TestOSRegistryOrderingIsStable is the regression test for PRODENG-3594.
//
// rig's registry promotes the winning resolver to the front of the list after
// every successful match. With rig's DefaultRegistry that is unsafe, because
// ResolveLinuxCompat matches any Linux host and sits ahead of ResolveLinux once
// a Windows host has been resolved. A mixed Linux/Windows cluster then reports
// Linux hosts as ID "linux", which resolves to no configurer at all.
//
// Detecting a Windows host must not change how the next Linux host is detected.
func TestOSRegistryOrderingIsStable(t *testing.T) {
	registry := osReleaseRegistry()

	// Baseline: a Linux host on its own is identified correctly.
	before, err := registry.Get(linuxRunner())
	require.NoError(t, err)
	require.Equal(t, "ubuntu", before.ID)
	require.Equal(t, "22.04", before.Version)

	// Resolve a Windows host. This is what reorders the registry: ResolveWindows
	// is promoted to index 0, displacing ResolveLinux and leaving
	// ResolveLinuxCompat ahead of it.
	win := &fakeRunner{windows: true, out: map[string]string{
		ps.Cmd("Get-CimInstance -ClassName Win32_OperatingSystem | Select-Object Caption, Version | ConvertTo-Json"): `{"Caption":"Microsoft Windows Server 2025 Datacenter","Version":"10.0.26100"}`,
		ps.Cmd("$env:PROCESSOR_ARCHITECTURE"): "AMD64",
	}}
	winRel, err := registry.Get(win)
	require.NoError(t, err, "fake Windows host must resolve, otherwise this test cannot reproduce the reordering")
	require.Equal(t, "windows", winRel.ID)

	// The Linux host must still be identified exactly as before.
	after, err := registry.Get(linuxRunner())
	require.NoError(t, err)
	require.Equal(t, "ubuntu", after.ID,
		"Linux host misidentified after a Windows host was resolved: the compat fallback won the ordering race")
	require.Equal(t, "22.04", after.Version)
}

// TestOSRegistryRejectsUnreadableOSRelease pins the other half of the fix: a
// host whose os-release cannot be read must fail loudly rather than degrade to
// the compat resolver's ID "linux", which no configurer matches. rig v0 errored
// here; the migration must not quietly lose that.
func TestOSRegistryRejectsUnreadableOSRelease(t *testing.T) {
	bare := &fakeRunner{out: map[string]string{
		"uname | grep -q Linux":               "",
		"uname -m":                            "x86_64",
		"command -v apt-get > /dev/null 2>&1": "",
	}}

	rel, err := osReleaseRegistry().Get(bare)
	require.ErrorIs(t, err, rigos.ErrNotRecognized)
	require.Nil(t, rel)
}
