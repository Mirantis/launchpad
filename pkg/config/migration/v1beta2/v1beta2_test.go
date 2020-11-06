package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMigration(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: UCP
spec:
  engine:
    installURL: http://get.example.com/
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b3 := []byte(`apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  engine:
    installURLLinux: http://get.example.com/
  hosts:
  - address: 10.0.0.1
    role: manager
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(b2, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, b3, out)
}
