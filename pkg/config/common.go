package config

import "os"

// EnsureDir ensures the given directory path exists, if not it will create the full path
// TODO This is slightly in wrong package now, but requires bit of refactoring to avoid
// cyclic imports with config --> util --> config
func EnsureDir(dirPath string) error {
	if _, serr := os.Stat(dirPath); os.IsNotExist(serr) {
		merr := os.MkdirAll(dirPath, os.ModePerm)
		if merr != nil {
			return merr
		}
	}
	return nil
}
