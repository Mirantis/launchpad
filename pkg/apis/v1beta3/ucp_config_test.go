package v1beta3

import (
	"testing"

	"github.com/Mirantis/mcc/pkg/util"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

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
