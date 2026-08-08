package configurer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
)

// TestMCRPackages covers the package set every Linux configurer installs.
//
// The runtime package lists the CLI only as a Recommends (a weak dependency),
// not a Requires/Depends, so the default must stay runtime-only: adding the CLI
// unconditionally would change what every existing cluster installs.
// spec.mcr.installCLI opts in for hosts where weak-dependency resolution is
// disabled or otherwise cannot be relied on. See PRODENG-3641.
func TestMCRPackages(t *testing.T) {
	t.Run("default installs the runtime only", func(t *testing.T) {
		require.Equal(t, []string{"docker-ee"}, configurer.MCRPackages(commonconfig.MCRConfig{}))
	})

	t.Run("installCLI adds the cli package", func(t *testing.T) {
		require.Equal(t, []string{"docker-ee", "docker-ee-cli"},
			configurer.MCRPackages(commonconfig.MCRConfig{InstallCLI: true}))
	})

	t.Run("both packages are returned together for a single transaction", func(t *testing.T) {
		// Returned as one slice, and the runtime stays first, so callers hand both
		// to the package manager in one invocation. Installing them separately
		// would allow the runtime and CLI to resolve to mismatched versions.
		pkgs := configurer.MCRPackages(commonconfig.MCRConfig{InstallCLI: true})
		require.Len(t, pkgs, 2)
		require.Equal(t, "docker-ee", pkgs[0])
	})
}
