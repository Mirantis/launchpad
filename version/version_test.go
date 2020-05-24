package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsProduction(t *testing.T) {
	require.False(t, IsProduction())

	Environment = "production"
	require.True(t, IsProduction())
}
