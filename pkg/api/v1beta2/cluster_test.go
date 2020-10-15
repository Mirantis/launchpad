package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, Migrate(&b2))
	require.Equal(t, b3, b2)
}
