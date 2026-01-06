package config

import (
	"testing"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMSRConfig_UseLegacyImageRepo(t *testing.T) {
	cfg := MSRConfig{}
	// >=3.1.15 || >=3.2.8 || >=3.3.2 is "mirantis"
	legacyVersions := []string{
		"2.8.1",
		"2.7.7",
		"2.6.14",
		"2.6.14-rc1",
		"2.5.2",
		"1.2.3",
	}
	modernVersions := []string{
		"2.8.2",
		"2.9.3",
		"2.7.8",
		"2.6.15",
		"2.6.15-rc5",
		"4.0.0",
	}

	for _, vs := range legacyVersions {
		v, _ := version.NewVersion(vs)
		require.True(t, cfg.UseLegacyImageRepo(v), "should be true for %s", vs)
	}

	for _, vs := range modernVersions {
		v, _ := version.NewVersion(vs)
		require.False(t, cfg.UseLegacyImageRepo(v), "should be false for %s", vs)
	}
}

func TestMSRConfig_LegacyDefaultVersionRepo(t *testing.T) {
	cfg := MSRConfig{}
	err := yaml.Unmarshal([]byte("version: 2.8.1"), &cfg)
	require.NoError(t, err)
	require.Equal(t, constant.ImageRepoLegacy, cfg.ImageRepo)
}

func TestMSRConfig_ModernDefaultVersionRepo(t *testing.T) {
	cfg := MSRConfig{}
	err := yaml.Unmarshal([]byte("version: 2.8.2"), &cfg)
	require.NoError(t, err)
	require.Equal(t, constant.ImageRepo, cfg.ImageRepo)
}

func TestMSRConfig_CustomRepo(t *testing.T) {
	cfg := MSRConfig{}
	err := yaml.Unmarshal([]byte("version: 2.8.2\nimageRepo: foo.foo/foo"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo.foo/foo", cfg.ImageRepo)
	cfg = MSRConfig{}
	err = yaml.Unmarshal([]byte("version: 2.8.1\nimageRepo: foo.foo/foo"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo.foo/foo", cfg.ImageRepo)
}
