package api

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestSwarmInstallFlags(t *testing.T) {
	cfg := MCRConfig{}
	err := yaml.Unmarshal([]byte("swarmInstallFlags:\n  - --foo=foofoo\n  - --bar barbar\n  - --foobar"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "--foobar", cfg.SwarmInstallFlags[2])
	require.Equal(t, 0, cfg.SwarmInstallFlags.Index("--foo"))
	require.Equal(t, 1, cfg.SwarmInstallFlags.Index("--bar"))
	require.Equal(t, 2, cfg.SwarmInstallFlags.Index("--foobar"))
	require.True(t, cfg.SwarmInstallFlags.Include("--bar"))
}

func TestSwarmUpdateCommands(t *testing.T) {
	cfg := MCRConfig{}
	err := yaml.Unmarshal([]byte("swarmUpdateCommands:\n  - command1\n  - command2\n  - command3"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "command3", cfg.SwarmUpdateCommands[2])
	require.Equal(t, 0, slices.Index(cfg.SwarmUpdateCommands, "command1"))
	require.Equal(t, 1, slices.Index(cfg.SwarmUpdateCommands, "command2"))
	require.Equal(t, 2, slices.Index(cfg.SwarmUpdateCommands, "command3"))
}
