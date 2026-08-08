package configurer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Mirantis/launchpad/pkg/configurer"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
)

// TestMCRInstallCommand pins the install command for every Linux family, with
// and without installRecommends.
//
// docker-ee declares docker-ee-cli and cri-dockerd-ee as recommended packages
// rather than hard requirements. Package managers install recommended packages
// by default, so the unset case must stay exactly what launchpad ran before:
// turning recommends on for everyone would change what every existing cluster
// installs. The set case must pass each manager's own opt-in. See PRODENG-3641.
func TestMCRInstallCommand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		manager  configurer.PackageManager
		recommen bool
		expected string
	}{
		{
			name:     "yum default is unchanged from the previous behaviour",
			manager:  configurer.Yum,
			expected: "yum install -y docker-ee",
		},
		{
			name:     "yum opts in with setopt",
			manager:  configurer.Yum,
			recommen: true,
			expected: "yum install -y --setopt=install_weak_deps=True docker-ee",
		},
		{
			name:     "apt-get default is unchanged from the previous behaviour",
			manager:  configurer.AptGet,
			expected: "DEBIAN_FRONTEND=noninteractive apt-get install -y -q docker-ee",
		},
		{
			name:     "apt-get opts in with the Install-Recommends config item",
			manager:  configurer.AptGet,
			recommen: true,
			expected: "DEBIAN_FRONTEND=noninteractive apt-get install -y -q -o APT::Install-Recommends=true docker-ee",
		},
		{
			// --allow-vendor-change must survive in both cases: without it zypper
			// cancels non-interactively on SLES cloud images. See PRODENG-3623.
			name:     "zypper keeps allow-vendor-change when recommends is unset",
			manager:  configurer.Zypper,
			expected: "zypper -n install -y --allow-vendor-change docker-ee",
		},
		{
			name:     "zypper opts in with --recommends alongside allow-vendor-change",
			manager:  configurer.Zypper,
			recommen: true,
			expected: "zypper -n install -y --allow-vendor-change --recommends docker-ee",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := configurer.MCRInstallCommand(tc.manager,
				commonconfig.MCRConfig{InstallRecommends: tc.recommen})
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

// TestMCRInstallCommandUnknownManager covers the error path, so an unhandled
// family fails loudly rather than returning an empty command that would be
// executed as a no-op.
func TestMCRInstallCommandUnknownManager(t *testing.T) {
	got, err := configurer.MCRInstallCommand(configurer.PackageManager(99), commonconfig.MCRConfig{})
	require.ErrorIs(t, err, configurer.ErrUnknownPackageManager)
	require.Empty(t, got)
}
