package helm

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChartNeedsUpgrade(t *testing.T) {
	h := NewHelmTestClient(t)
	rd, _ := InstallCertManagerChart(t, h)

	rdChartVersion := rd.Version

	testCases := []string{
		// ChartNeedsUpgrade returns true if the version is DIFFERENT from
		// the installed one, so anything but Version should return
		// true. It's written this way to support downgrades.
		rdChartVersion,
		"1.0.0",
		"1.0.1",
		"1.0.100",
		"2.1.2",
		"3.0.0-alpha",
		"3.0.0-alpha.1",
		"0.9.9",
		"0.9.0",
		"0.0.1",
		"0.1.1-beta-someothertext",
	}

	for _, tc := range testCases {
		vers, err := version.NewVersion(tc)
		require.NoError(t, err)

		actual, err := h.ChartNeedsUpgrade(rd.ReleaseName, vers)
		if assert.NoError(t, err) {
			if tc == rdChartVersion {
				assert.False(t, actual, "version: %s does match current version: %s", tc, rdChartVersion)
			} else {
				assert.True(t, actual, "version: %s does not match current version: %s", tc, rdChartVersion)
			}
		}
	}
}
