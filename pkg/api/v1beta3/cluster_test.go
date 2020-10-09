package v1beta3

import (
	"testing"

	"github.com/stretchr/testify/require"
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
	b4 := []byte(`apiVersion: launchpad.mirantis.com/v1
kind: DockerEnterprise
spec:
  hosts:
  - address: 10.0.0.1
    role: manager
`)
	require.NoError(t, Migrate(&b3))
	require.Equal(t, b4, b3)
}
