package byteutil

import (
	"fmt"
)

// FormatBytes formats a number of bytes into something like "200 KiB".
func FormatBytes(bytes uint64) string {
	float := float64(bytes)
	units := []string{
		"bytes",
		"KiB",
		"MiB",
		"GiB",
		"TiB",
	}
	logBase1024 := 0
	for float >= 1024.0 && logBase1024 < len(units) {
		float /= 1024.0
		logBase1024++
	}
	return fmt.Sprintf("%d %s", uint64(float), units[logBase1024])
}
