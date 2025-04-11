package docker_test

import (
	"testing"

	"github.com/Mirantis/launchpad/pkg/docker"
	"github.com/stretchr/testify/require"
)

func Test_CompareRepos(t *testing.T) {
	require.True(t, docker.CompareRepos("registry.mirantis.com/mirantis", "registry.mirantis.com/mirantis"))
	require.False(t, docker.CompareRepos("registry.mirantis.com/mirantis", "registry.ci.mirantis.com/mirantiseng"))

	require.True(t, docker.CompareRepos("", ""))

	// some case handling
	require.True(t, docker.CompareRepos("registry.Mirantis.com/mirantis", "registry.mirantis.com/Mirantis"))
	require.True(t, docker.CompareRepos("Registry.mirantis.com/mirantis", "registry.mirantis.com/mirantis"))
	require.True(t, docker.CompareRepos("registry.mirantis.com/mirantis", "registry.mirantis.com/Mirantis"))

	// the docker.io special case
	require.True(t, docker.CompareRepos("docker.io/mirantis", ""))
}
