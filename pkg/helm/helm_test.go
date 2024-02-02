package helm

import (
	"context"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Mirantis/mcc/pkg/constant"
)

func TestChartNeedsUpgrade(t *testing.T) {
	h, _ := NewHelmTestClient(t, nil)
	InstallRethinkDBOperatorChart(t, h)

	testCases := []struct {
		version              string
		expectedNeedsUpgrade bool
	}{
		// ChartNeedsUpgrade returns true if the version is DIFFERENT from
		// the installed one, so anything but "1.0.0" should return true.
		// It's written this way to support downgrades.
		{"1.0.0", false},
		{"1.0.1", true},
		{"1.0.100", true},
		{"2.1.2", true},
		{"3.0.0-alpha", true},
		{"3.0.0-alpha.1", true},
		{"0.9.9", true},
		{"0.9.0", true},
		{"0.0.1", true},
		{"0.1.1-beta-someothertext", true},
	}

	for _, tc := range testCases {
		vers, err := version.NewVersion(tc.version)
		require.NoError(t, err)

		actual, err := h.ChartNeedsUpgrade(context.Background(), constant.RethinkDBOperator, vers)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedNeedsUpgrade, actual, "version: %s", tc.version)
	}
}
