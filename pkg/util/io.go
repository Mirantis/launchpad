package util

import (
	"fmt"
	"os"

	"github.com/mitchellh/go-homedir"
)

// EnsureDir ensures the given directory path exists, if not it will create the full path.
func EnsureDir(dirPath string) error {
	if _, serr := os.Stat(dirPath); os.IsNotExist(serr) {
		merr := os.MkdirAll(dirPath, os.ModePerm)
		if merr != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, merr)
		}
	}
	return nil
}

// LoadExternalFile helper for reading data from references to external files.
var LoadExternalFile = func(path string) ([]byte, error) {
	realpath, err := homedir.Expand(path)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to expand path %s: %w", path, err)
	}

	filedata, err := os.ReadFile(realpath)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read file %s: %w", realpath, err)
	}
	return filedata, nil
}

// FormatBytes formats a number of bytes into something like "200 KiB".
func FormatBytes(bytes uint64) string {
	floatBytes := float64(bytes)
	units := []string{
		"bytes",
		"KiB",
		"MiB",
		"GiB",
	}
	logBase1024 := 0
	for floatBytes > 1024.0 && logBase1024 < len(units) {
		floatBytes /= 1024.0
		logBase1024++
	}
	return fmt.Sprintf("%d %s", uint64(floatBytes), units[logBase1024])
}
