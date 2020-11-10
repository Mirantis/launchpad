package api

import (
	"testing"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestUcpConfigFlags(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("installFlags:\n  - --foo=foofoo\n  - --san foo\n  - --ucp-insecure-tls"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "--ucp-insecure-tls", cfg.InstallFlags[2])
	require.Equal(t, 0, cfg.InstallFlags.Index("--foo"))
	require.Equal(t, 1, cfg.InstallFlags.Index("--san"))
	require.Equal(t, 2, cfg.InstallFlags.Index("--ucp-insecure-tls"))
	require.True(t, cfg.InstallFlags.Include("--san"))

	cfg.InstallFlags.Delete("--san")
	require.Equal(t, 1, cfg.InstallFlags.Index("--ucp-insecure-tls"))
	require.False(t, cfg.InstallFlags.Include("--san"))

	cfg.InstallFlags.AddOrReplace("--san 10.0.0.1")
	require.Equal(t, 2, cfg.InstallFlags.Index("--san"))
	require.Equal(t, "--san 10.0.0.1", cfg.InstallFlags.Get("--san"))
	require.Equal(t, "10.0.0.1", cfg.InstallFlags.GetValue("--san"))
	require.Equal(t, "foofoo", cfg.InstallFlags.GetValue("--foo"))

	require.Len(t, cfg.InstallFlags, 3)
	cfg.InstallFlags.AddOrReplace("--bar=barbar")
	require.Equal(t, 3, cfg.InstallFlags.Index("--bar"))
	require.Equal(t, "barbar", cfg.InstallFlags.GetValue("--bar"))

	require.Len(t, cfg.InstallFlags, 4)
	cfg.InstallFlags.AddUnlessExist("--bar=borbor")
	require.Len(t, cfg.InstallFlags, 4)
	require.Equal(t, "barbar", cfg.InstallFlags.GetValue("--bar"))

	cfg.InstallFlags.AddUnlessExist("--help")
	require.Len(t, cfg.InstallFlags, 5)
	require.True(t, cfg.InstallFlags.Include("--help"))
}

func TestUcpConfig_YAML_ConfigData(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("configData: abcd"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "abcd", cfg.ConfigData)
}

func TestUcpConfig_YAML_ConfigFile(t *testing.T) {
	cfg := UcpConfig{}
	util.LoadExternalFile = func(path string) ([]byte, error) {
		return []byte("abcd"), nil
	}
	err := yaml.Unmarshal([]byte("configFile: test_path.toml"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "abcd", cfg.ConfigData)
}

func TestUcpConfig_UseLegacyImageRepo(t *testing.T) {
	cfg := UcpConfig{}
	// >=3.1.15 || >=3.2.8 || >=3.3.2 is "mirantis"
	legacyVersions := []string{
		"3.1.14",
		"3.2.7",
		"3.3.1",
		"2.0.0",
		"3.2.7-tp7",
	}
	modernVersions := []string{
		"3.1.15",
		"3.1.16-rc1",
		"3.2.8",
		"3.3.2",
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

func TestUcpConfig_LegacyDefaultVersionRepo(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("version: 3.2.1"), &cfg)
	require.NoError(t, err)
	require.Equal(t, constant.ImageRepoLegacy, cfg.ImageRepo)
}

func TestUcpConfig_ModernDefaultVersionRepo(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("version: 3.2.8"), &cfg)
	require.NoError(t, err)
	require.Equal(t, constant.ImageRepo, cfg.ImageRepo)
}

func TestUcpConfig_CustomRepo(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("version: 3.2.7\nimageRepo: foo.foo/foo"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo.foo/foo", cfg.ImageRepo)
}

func TestUcpConfig_Credentials(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("adminUsername: foo\nadminPassword: bar\n"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo", cfg.AdminUsername)
	require.Equal(t, "bar", cfg.AdminPassword)
}

func TestUcpConfig_CredentialsFromInstallFlags(t *testing.T) {
	cfg := UcpConfig{}
	err := yaml.Unmarshal([]byte("installFlags:\n  - --admin-username=\"foo\"\n  - --admin-password bar\n"), &cfg)
	require.NoError(t, err)
	require.Equal(t, "foo", cfg.AdminUsername)
	require.Equal(t, "bar", cfg.AdminPassword)
}
