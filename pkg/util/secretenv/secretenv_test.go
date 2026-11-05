package secretenv_test

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"

	"github.com/Mirantis/launchpad/pkg/util/secretenv"
	"github.com/k0sproject/rig/v2/remotefs"
	"github.com/stretchr/testify/require"
)

// multiLineSecret stands in for the PEM blocks MSR_KEY and MSR_CERT carry.
// Its point is the newlines: docker's own --env-file format is one
// KEY=VALUE per line and could not carry this.
const multiLineSecret = "fake-key-line-one\nfake-key-line-two\nfake-key-line-three\n"

// writtenFile records what reached the host.
type writtenFile struct {
	data    string
	mode    fs.FileMode
	removed bool
}

// fakeFS records writes and removals in place of a real host filesystem. Only
// the three calls secretenv makes are implemented; the rest of remotefs.FS is
// embedded so an unused call panics loudly rather than silently returning a
// zero value.
type fakeFS struct {
	remotefs.FS

	files     map[string]*writtenFile
	order     []string
	created   int
	createErr error
	writeErr  error
	removeErr error
}

func newFakeFS() *fakeFS {
	return &fakeFS{files: map[string]*writtenFile{}}
}

func (f *fakeFS) CreateTemp(_, prefix string) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}

	f.created++
	path := fmt.Sprintf("/tmp/%s%06d", prefix, f.created)
	f.files[path] = &writtenFile{}
	f.order = append(f.order, path)

	return path, nil
}

func (f *fakeFS) WriteFile(path string, data []byte, mode fs.FileMode) error {
	if f.writeErr != nil {
		return f.writeErr
	}

	file, ok := f.files[path]
	if !ok {
		return fmt.Errorf("write to a path that was never created: %s", path) //nolint:err113 // test helper
	}

	file.data = string(data)
	file.mode = mode

	return nil
}

func (f *fakeFS) Remove(path string) error {
	if f.removeErr != nil {
		return f.removeErr
	}

	file, ok := f.files[path]
	if !ok {
		return fmt.Errorf("remove of a path that was never created: %s", path) //nolint:err113 // test helper
	}

	file.removed = true

	return nil
}

func (f *fakeFS) live() []string {
	var live []string

	for _, path := range f.order {
		if !f.files[path].removed {
			live = append(live, path)
		}
	}

	return live
}

type fakeHost struct{ fs *fakeFS }

func (h fakeHost) FS() remotefs.FS { return h.fs }

func TestStageKeepsSecretsOutOfTheCommandLine(t *testing.T) {
	host := fakeHost{fs: newFakeFS()}

	vars, err := secretenv.Stage(host, map[string]string{
		"UCP_ADMIN_PASSWORD": "fake-admin-password",
		"MSR_KEY":            multiLineSecret,
	})
	require.NoError(t, err)

	command := vars.Assignments() + " docker run " + strings.Join(vars.PassFlags(), " ") + " mirantis/ucp install"

	require.NotContains(t, command, "fake-admin-password")
	require.NotContains(t, command, "fake-key-line-one")
	require.Contains(t, command, "-e UCP_ADMIN_PASSWORD")
	require.Contains(t, command, "-e MSR_KEY")
	// The value is read on the host when the command runs.
	require.Contains(t, command, `UCP_ADMIN_PASSWORD="$(cat -- /tmp/launchpad-env-000002)"`)
	require.Equal(t, []string{"MSR_KEY", "UCP_ADMIN_PASSWORD"}, vars.Names())

	// The multi-line PEM survives intact, which a docker --env-file could
	// not carry.
	require.Equal(t, multiLineSecret, host.fs.files["/tmp/launchpad-env-000001"].data)
}

func TestStageCreatesFilesPrivate(t *testing.T) {
	host := fakeHost{fs: newFakeFS()}

	_, err := secretenv.Stage(host, map[string]string{"UCP_ADMIN_PASSWORD": "fake-admin-password"})
	require.NoError(t, err)

	for path, file := range host.fs.files {
		require.Equalf(t, fs.FileMode(0o600), file.mode, "%s is readable by other users on the host", path)
	}
}

func TestRemoveDeletesEveryStagedFile(t *testing.T) {
	host := fakeHost{fs: newFakeFS()}

	vars, err := secretenv.Stage(host, map[string]string{
		"UCP_ADMIN_PASSWORD": "fake-admin-password",
		"UCP_LICENSE":        "fake-license",
		"MSR_KEY":            multiLineSecret,
	})
	require.NoError(t, err)
	require.Len(t, host.fs.live(), 3)

	require.NoError(t, vars.Remove())
	require.Empty(t, host.fs.live(), "a staged secret outlived the command")
}

// A failure part way through staging must not leave the values that were
// already written behind.
func TestStageCleansUpAfterAFailedWrite(t *testing.T) {
	host := fakeHost{fs: newFakeFS()}
	host.fs.writeErr = errors.New("disk full")

	_, err := secretenv.Stage(host, map[string]string{"UCP_ADMIN_PASSWORD": "fake-admin-password"})
	require.Error(t, err)
	require.Empty(t, host.fs.live(), "a staged secret survived a failed staging")
}

// Removal is attempted for every file even when one fails, and the failures
// are reported rather than swallowed.
func TestRemoveReportsFailures(t *testing.T) {
	host := fakeHost{fs: newFakeFS()}

	vars, err := secretenv.Stage(host, map[string]string{"UCP_ADMIN_PASSWORD": "fake-admin-password", "UCP_LICENSE": "fake-license"})
	require.NoError(t, err)

	host.fs.removeErr = errors.New("permission denied")

	err = vars.Remove()
	require.Error(t, err)
	require.Contains(t, err.Error(), "UCP_ADMIN_PASSWORD")
	require.Contains(t, err.Error(), "UCP_LICENSE")
}

func TestEmptySetProducesNoCommandText(t *testing.T) {
	host := fakeHost{fs: newFakeFS()}

	vars, err := secretenv.Stage(host, nil)
	require.NoError(t, err)

	require.Empty(t, vars.Assignments())
	require.Empty(t, vars.PassFlags())
	require.NoError(t, vars.Remove())
}

func TestExtractPullsCredentialFlagsOut(t *testing.T) {
	envByFlag := map[string]string{
		"--admin-password": "UCP_ADMIN_PASSWORD",
		"--ucp-password":   "UCP_PASSWORD",
	}

	values, kept := secretenv.Extract([]string{
		"--san=mke.example.com",
		"--admin-password='fake-admin-password'",
		`--ucp-password="fake-ucp-password"`,
		"--pull never",
	}, envByFlag)

	require.Equal(t, map[string]string{
		"UCP_ADMIN_PASSWORD": "fake-admin-password",
		"UCP_PASSWORD":       "fake-ucp-password",
	}, values)
	require.Equal(t, []string{"--san=mke.example.com", "--pull never"}, kept)
}

// A flag with no value carries no secret, and dropping it would change what
// the bootstrapper is asked to do.
func TestExtractLeavesValuelessAndUnknownFlags(t *testing.T) {
	values, kept := secretenv.Extract(
		[]string{"--admin-password", "--admin-password=", "--force-minimums", "--license-file /tmp/x"},
		map[string]string{"--admin-password": "UCP_ADMIN_PASSWORD"},
	)

	require.Empty(t, values)
	require.Equal(t, []string{"--admin-password", "--admin-password=", "--force-minimums", "--license-file /tmp/x"}, kept)
}
