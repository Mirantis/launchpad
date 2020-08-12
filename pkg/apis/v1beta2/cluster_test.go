package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationToV1Beta3Basic(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: UCP
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b3 := []byte(`apiVersion: launchpad.mirantis.com/v1beta3
kind: DockerEnterprise
spec:
  hosts:
  - address: 10.0.0.1
    role: manager
`)
	require.NoError(t, MigrateToV1Beta3(&b2))
	require.Equal(t, b3, b2)
}

func TestMigrationToV1Beta3WithDTR(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: UCP
spec:
  dtr:
    version: 1.2.3
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	require.EqualError(t, MigrateToV1Beta3(&b2), "dtr requires apiVersion >= launchpad.mirantis.com/v1beta3")
}

func TestMigrationToV1Beta3WithDockerEnterprise(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	require.EqualError(t, MigrateToV1Beta3(&b2), "kind: DockerEnterprise is only available in version >= 0.13")
}
