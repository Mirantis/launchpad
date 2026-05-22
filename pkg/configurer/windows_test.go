package configurer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWindowsInstallerVersion(t *testing.T) {
	tests := []struct {
		channel string
		want    string
	}{
		// Standard channels — always "latest"; docker-latest.zip is published
		{"stable-29.2", "latest"},
		{"stable-25.0", "latest"},
		{"ee-stable-29.2", "latest"},
		// FIPS channels — version extracted; only docker-<version>+fips.zip is published
		{"stable-29.2.1/fips", "29.2.1"},
		{"stable-29.4.1/fips", "29.4.1"},
		{"test-29.4.1/fips", "29.4.1"},
		// FIPS RC channel
		{"test-29.4.1-rc3/fips", "29.4.1-rc3"},
		// FIPS channel with no parseable version — falls back to latest
		{"stable/fips", "latest"},
	}
	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			require.Equal(t, tt.want, windowsInstallerVersion(tt.channel), "channel=%q", tt.channel)
		})
	}
}
