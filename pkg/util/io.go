package util

import "os"

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
