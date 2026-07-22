package config_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
)

func TestMCRConfig_ChannelRequired(t *testing.T) {
	// Channel must not be defaulted when absent — ClusterConfig.Validate()
	// catches an empty channel via the validate:"required" struct tag.
	cfg := commonconfig.MCRConfig{}
	err := yaml.Unmarshal([]byte("repoURL: https://example.com"), &cfg)
	require.NoError(t, err)
	require.Empty(t, cfg.Channel, "channel must not be defaulted when absent")
}

func TestSwarmInstallFlags(t *testing.T) {
	cfg := commonconfig.MCRConfig{}
	err := yaml.Unmarshal([]byte("channel: stable\nswarmInstallFlags:\n  - --foo=foofoo\n  - --bar barbar\n  - --foobar"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "--foobar", cfg.SwarmInstallFlags[2])
	require.Equal(t, 0, cfg.SwarmInstallFlags.Index("--foo"))
	require.Equal(t, 1, cfg.SwarmInstallFlags.Index("--bar"))
	require.Equal(t, 2, cfg.SwarmInstallFlags.Index("--foobar"))
	require.True(t, cfg.SwarmInstallFlags.Include("--bar"))
}

func TestSwarmUpdateCommands(t *testing.T) {
	cfg := commonconfig.MCRConfig{}
	err := yaml.Unmarshal([]byte("channel: stable\nswarmUpdateCommands:\n  - command1\n  - command2\n  - command3"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "command3", cfg.SwarmUpdateCommands[2])
	require.Equal(t, 0, slices.Index(cfg.SwarmUpdateCommands, "command1"))
	require.Equal(t, 1, slices.Index(cfg.SwarmUpdateCommands, "command2"))
	require.Equal(t, 2, slices.Index(cfg.SwarmUpdateCommands, "command3"))
}
