package v13

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestApiVersionMigration(t *testing.T) {
	v13 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.3
kind: mke
spec:
  hosts:
  - ssh:
      address: 10.0.0.1
      port: 22
  - ssh:
      address: 10.0.0.2
  - winRM:
      address: 10.0.0.3
      port: 2000
  - localhost:
      enabled: true
`)
	// go's YAML marshal does not add the --- header
	v14 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  hosts:
  - ssh:
      address: 10.0.0.1
      port: 22
  - ssh:
      address: 10.0.0.2
  - winRM:
      address: 10.0.0.3
      port: 2000
  - localhost:
      enabled: true
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v13, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v14), string(out))
}
