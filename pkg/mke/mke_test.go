package mke

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
)

func TestVersionGreaterThan(t *testing.T) {
	pairs := [][]string{
		{"1.0.0-beta2", "1.0.0-beta1"},
		{"1.0.0", "1.0.0-rc1"},
		{"1.0.0-rc1", "1.0.0-tp3"},
	}

	for _, pair := range pairs {
		va, _ := version.NewVersion(pair[0])
		vb, _ := version.NewVersion(pair[1])
		require.Truef(t, VersionGreaterThan(va, vb), "%s should be greater than %s", va, vb)
	}
}
