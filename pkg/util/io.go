package util

import (
	"io/ioutil"
	"os"

	"github.com/mitchellh/go-homedir"
)

// EnsureDir ensures the given directory path exists, if not it will create the full path
func EnsureDir(dirPath string) error {
	if _, serr := os.Stat(dirPath); os.IsNotExist(serr) {
		merr := os.MkdirAll(dirPath, os.ModePerm)
		if merr != nil {
			return merr
		}
	}
	return nil
}

// LoadExternalFile helper for reading data from references to external files
var LoadExternalFile = func(path string) ([]byte, error) {
	realpath, err := homedir.Expand(path)
	if err != nil {
		return []byte{}, err
	}

	filedata, err := ioutil.ReadFile(realpath)
	if err != nil {
		return []byte{}, err
	}
	return filedata, nil
}
