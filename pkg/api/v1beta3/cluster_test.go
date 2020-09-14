package v1beta3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationToCurrent(t *testing.T) {
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
	require.NoError(t, MigrateToCurrent(&b3))
	require.Equal(t, b4, b3)
}

func TestMigrationToCurrentWithHooks(t *testing.T) {
	b3 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta3"
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
      hooks:
        apply:
          before:
            - ls -al
`)
	require.EqualError(t, MigrateToCurrent(&b3), "host hooks require apiVersion >= launchpad.mirantis.com/v1")
}

func TestMigrationToCurrentWithLocalhost(t *testing.T) {
	b3 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta3"
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
      localhost: true
`)
	require.EqualError(t, MigrateToCurrent(&b3), "localhost connection requires apiVersion >= launchpad.mirantis.com/v1")
}
