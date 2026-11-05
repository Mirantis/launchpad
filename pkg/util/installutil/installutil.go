package installutil

import (
	"fmt"
	"os"
	"strings"
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

	// TrimSpace rather than shell quoting: the licence no longer reaches a
	// command line, so there is nothing for a shell to parse, but the
	// documented form "--license $(cat license.lic)" had its surrounding
	// whitespace removed by the shell and the value delivered here should
	// match it. PRODENG-3920 added the trim; this keeps it.
	return strings.TrimSpace(string(license)), nil
}
