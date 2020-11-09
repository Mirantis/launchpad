package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMigration(t *testing.T) {
	b1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta1"
kind: UCP
spec:
  hosts:
    - address: "10.0.0.1"
      sshKeyPath: /tmp/tmp
      sshPort: 9022
      user: "admin"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b2 := []byte(`apiVersion: launchpad.mirantis.com/v1beta2
kind: UCP
spec:
  hosts:
  - address: 10.0.0.1
    role: manager
    ssh:
      keyPath: /tmp/tmp
      port: 9022
      user: admin
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(b1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, b2, out)
}
