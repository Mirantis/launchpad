package v1beta3

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMigration(t *testing.T) {
	b3 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta3"
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	v1 := []byte(`apiVersion: launchpad.mirantis.com/v1
kind: DockerEnterprise
spec:
  hosts:
  - address: 10.0.0.1
    role: manager
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(b3, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, v1, out)
}
