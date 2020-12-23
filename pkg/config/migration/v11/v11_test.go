package v11

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestEngineToMCRMigration(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1.1"
kind: mke
spec:
  engine:
    version: 10.0.1
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  mcr:
    version: 10.0.1
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}

func TestHostEngineConfigToMCRConfigMigration(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1.1"
kind: mke
spec:
  hosts:
    - address: 10.0.0.1
      engineConfig:
        debug: true
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.2
kind: mke
spec:
  hosts:
  - address: 10.0.0.1
    mcrConfig:
      debug: true
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}
