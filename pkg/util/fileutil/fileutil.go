package fileutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LoadExternalFile helper for reading data from references to external files.
var LoadExternalFile = func(path string) ([]byte, error) {
	realpath, err := ExpandHomeDir(path)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to expand home dir: %w", err)
	}

	filedata, err := os.ReadFile(realpath) // #nosec G304 G703 -- path is user-provided by design (CLI/config file reference)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read %q: %w", realpath, err)
	}
	return filedata, nil
}

var errCannotExpandHomeDir = errors.New("cannot expand user-specific home dir")

// ExpandHomeDir expands the path to include the home directory if the path
// is prefixed with `~`. If it isn't prefixed with `~`, the path is
// returned as-is.
func ExpandHomeDir(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", errCannotExpandHomeDir
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir: %w", err)
	}

	return filepath.Join(dir, path[1:]), nil
}

// MkdirPrivate creates dir and every missing parent, then narrows dir itself
// to 0700 even when it already existed.
//
// os.MkdirAll applies its mode only to the directories it creates, so without
// the explicit Chmod an upgrade over a tree created by an earlier release
// keeps that tree's permissions and a tightened mode never reaches an existing
// installation.
//
// Limit: only dir itself is narrowed, not intermediate parents. Creating
// "a/b" when "a" already exists at 0777 leaves "a" at 0777 and narrows only
// "b". Callers that need a whole chain narrowed must call this for each
// level. In launchpad this is defence in depth rather than exposure, because
// the roots involved -- the bundle directory and the per-cluster state
// directory -- are themselves narrowed by an earlier call, so group and other
// cannot traverse in to reach an intermediate.
func MkdirPrivate(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("failed to restrict directory %q: %w", dir, err)
	}
	return nil
}

// WritePrivateFile writes data to path at 0600, narrowing the file even when
// it already existed.
//
// os.WriteFile applies its mode only when it creates the file, so refreshing a
// file written by an earlier release would otherwise leave it at that
// release's mode. The descriptor is truncated and chmodded before any data is
// written, so what is briefly visible under a previous looser mode is an empty
// file, never contents.
//
// Limit: POSIX checks permission at open, so a process that already held the
// file open before the chmod keeps its access and will see the new contents.
// Closing that would require writing to a temporary file and renaming, to get
// a new inode.
func WritePrivateFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- callers pass validated paths
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer file.Close()

	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("failed to restrict %q: %w", path, err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close %q: %w", path, err)
	}
	return nil
}
