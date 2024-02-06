package fileutil

import (
	"errors"
	"os"
	"path/filepath"
)

// LoadExternalFile helper for reading data from references to external files.
var LoadExternalFile = func(path string) ([]byte, error) {
	realpath, err := Expand(path)
	if err != nil {
		return []byte{}, err
	}

	filedata, err := os.ReadFile(realpath)
	if err != nil {
		return []byte{}, err
	}
	return filedata, nil
}

// Expand  expands the path to include the home directory if the path
// is prefixed with `~`. If it isn't prefixed with `~`, the path is
// returned as-is.
func Expand(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", errors.New("cannot expand user-specific home dir")
	}

	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, path[1:]), nil
}
