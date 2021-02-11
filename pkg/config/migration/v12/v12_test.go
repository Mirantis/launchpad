package v12

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestAddressMigration(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1.2"
kind: mke
spec:
  hosts:
    - address: 10.0.0.1
      ssh:
        port: 22
    - address: 10.0.0.2
    - address: 10.0.0.3
      winRM:
        port: 2000
    - address: 10.0.0.4
      localhost: true
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.3
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
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}
