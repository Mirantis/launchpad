package fileutil

import (
	"os"

	"github.com/mitchellh/go-homedir"
)

// LoadExternalFile helper for reading data from references to external files.
var LoadExternalFile = func(path string) ([]byte, error) {
	realpath, err := homedir.Expand(path)
	if err != nil {
		return []byte{}, err
	}

	filedata, err := os.ReadFile(realpath)
	if err != nil {
		return []byte{}, err
	}
	return filedata, nil
}
