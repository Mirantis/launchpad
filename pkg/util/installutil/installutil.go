package installutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/k0sproject/rig/v2/sh/shellescape"
)

// Shared install utility between install phases for different products
// SetupLicenseFile reads the license file and returns a license string command
// flag to be used with MSR and MKE installers.
func SetupLicenseFile(licenseFilePath string) (string, error) {
	license, err := os.ReadFile(licenseFilePath) // #nosec G304 G703 -- path is user-provided by design (CLI flag)
	if err != nil {
		return "", fmt.Errorf("failed to read license file: %w", err)
	}

	// The license is embedded in a command line that a remote login shell
	// executes; the whole flag value must survive as a single shell word.
	licenseFlag := fmt.Sprintf("--license %s", shellescape.Quote(strings.TrimSpace(string(license))))

	return licenseFlag, nil
}
