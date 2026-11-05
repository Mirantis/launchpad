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
func MkdirPrivate(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("failed to restrict directory %q: %w", dir, err)
	}
	return nil
}
