package installutil

import (
	"fmt"
	"os"
)

// Shared install utility between install phases for different products
// SetupLicenseFile reads the license file and returns a license string command
// flag to be used with MSR and MKE installers.
func SetupLicenseFile(licenseFilePath string) (string, error) {
	license, err := os.ReadFile(licenseFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read license file: %w", err)
	}
	licenseFlag := fmt.Sprintf("--license '%s'", string(license))
	return licenseFlag, nil
}
