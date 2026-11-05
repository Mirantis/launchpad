package installutil

import (
	"fmt"
	"os"
)

// Shared install utility between install phases for different products.
//
// ReadLicense reads a license file and returns its contents. The value is a
// credential and is handed to the installers as an environment value, never
// as a command line argument, so this returns the licence itself rather than
// a formatted flag.
func ReadLicense(licenseFilePath string) (string, error) {
	license, err := os.ReadFile(licenseFilePath) // #nosec G304 G703 -- path is user-provided by design (CLI flag)
	if err != nil {
		return "", fmt.Errorf("failed to read license file: %w", err)
	}

	return string(license), nil
}
