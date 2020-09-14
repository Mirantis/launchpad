package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationToCurrent(t *testing.T) {
	b1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta1"
kind: UCP
spec:
  engine:
    installURL: http://get.example.com/
  hosts:
    - address: "10.0.0.1"
      sshKeyPath: /tmp/tmp
      sshPort: 9022
      user: "admin"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b2 := []byte(`apiVersion: launchpad.mirantis.com/v1
kind: DockerEnterprise
spec:
  engine:
    installURLLinux: http://get.example.com/
  hosts:
  - address: 10.0.0.1
    role: manager
    ssh:
      keyPath: /tmp/tmp
      port: 9022
      user: admin
`)
	require.NoError(t, MigrateToCurrent(&b1))
	require.Equal(t, b2, b1)
}

func TestMigrationToCurrentNoInstallURL(t *testing.T) {
	b1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta1"
kind: UCP
spec:
  engine:
    version: 1.2.3
  hosts:
    - address: "10.0.0.1"
      sshKeyPath: /tmp/tmp
      sshPort: 9022
      user: "admin"
      role: "manager"
`)
	// go's YAML marshal does not add the --- header
	b2 := []byte(`apiVersion: launchpad.mirantis.com/v1
kind: DockerEnterprise
spec:
  engine:
    version: 1.2.3
  hosts:
  - address: 10.0.0.1
    role: manager
    ssh:
      keyPath: /tmp/tmp
      port: 9022
      user: admin
`)
	require.NoError(t, MigrateToCurrent(&b1))
	require.Equal(t, b2, b1)
}

func TestMigrationToCurrentWithDockerEnterprise(t *testing.T) {
	b1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta1"
kind: DockerEnterprise
spec:
  hosts:
    - address: "10.0.0.1"
      role: "manager"
`)
	require.EqualError(t, MigrateToCurrent(&b1), "kind: DockerEnterprise is only available in version >= 0.13")
}

func TestMigrationToCurrentWithHooks(t *testing.T) {
	b1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta1"
kind: UCP
spec:
  hosts:
    - address: "10.0.0.1"
      hooks:
        apply:
          before:
            - ls -al
      role: "manager"
`)
	require.EqualError(t, MigrateToCurrent(&b1), "host hooks require apiVersion >= launchpad.mirantis.com/v1")
}

func TestMigrationToCurrentWithLocalhost(t *testing.T) {
	b1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1beta1"
kind: UCP
spec:
  hosts:
    - address: "10.0.0.1"
      localhost: true
      role: "manager"
`)
	require.EqualError(t, MigrateToCurrent(&b1), "localhost connection requires apiVersion >= launchpad.mirantis.com/v1")
}
