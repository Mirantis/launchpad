package v1beta1

import (
	"io/ioutil"

	"github.com/mitchellh/go-homedir"
)

// Helper for reading data from references to external files
var loadExternalFile = func(path string) ([]byte, error) {
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
