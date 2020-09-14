package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationToCurrent(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: UCP
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b3 := []byte(`apiVersion: launchpad.mirantis.com/v1beta4
kind: DockerEnterprise
spec:
  hosts:
  - address: 10.0.0.1
    role: manager
`)
	require.NoError(t, MigrateToCurrent(&b2))
	require.Equal(t, b3, b2)
}

func TestMigrationToCurrentWithDTR(t *testing.T) {
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
	require.EqualError(t, MigrateToCurrent(&b2), "dtr requires apiVersion >= launchpad.mirantis.com/v1beta3")
}

func TestMigrationToCurrentWithDockerEnterprise(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	require.EqualError(t, MigrateToCurrent(&b2), "kind: DockerEnterprise is only available in version >= 0.13")
}

func TestMigrationToCurrentWithHooks(t *testing.T) {
	b2 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta2"
kind: UCP
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
      hooks:
        apply:
          before:
            - ls -al
`)
	require.EqualError(t, MigrateToCurrent(&b2), "host hooks require apiVersion >= launchpad.mirantis.com/v1beta4")
}
