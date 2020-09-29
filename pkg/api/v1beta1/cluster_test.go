package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
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
	b2 := []byte(`apiVersion: launchpad.mirantis.com/v1beta2
kind: UCP
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
	require.NoError(t, Migrate(&b1))
	require.Equal(t, b2, b1)
}

func TestMigrationNoInstallURL(t *testing.T) {
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
	b2 := []byte(`apiVersion: launchpad.mirantis.com/v1beta2
kind: UCP
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
	require.NoError(t, Migrate(&b1))
	require.Equal(t, b2, b1)
}
