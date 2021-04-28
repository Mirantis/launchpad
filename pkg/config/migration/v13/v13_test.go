package v13

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestVersionMigration(t *testing.T) {
	v13 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.3
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      port: 22
  - role: worker
    ssh:
      address: 10.0.0.2
  - role: msr
    ssh:
      address: 10.0.0.3
`)
	// go's YAML marshal does not add the --- header
	v14 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      port: 22
  - role: worker
    ssh:
      address: 10.0.0.2
  - role: msr
    ssh:
      address: 10.0.0.3
  mke:
    version: 3.4.0
  msr:
    version: 2.9.0
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v13, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v14), string(out))
}

func TestVersionMigrationNoOverwrite(t *testing.T) {
	v13 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.3
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      port: 22
  - role: worker
    ssh:
      address: 10.0.0.2
  - role: msr
    ssh:
      address: 10.0.0.3
  mke:
    version: 3.3.7
  msr:
    version: 2.8.5
`)
	// go's YAML marshal does not add the --- header
	v14 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.4
kind: mke
spec:
  hosts:
  - role: manager
    ssh:
      address: 10.0.0.1
      port: 22
  - role: worker
    ssh:
      address: 10.0.0.2
  - role: msr
    ssh:
      address: 10.0.0.3
  mke:
    version: 3.3.7
  msr:
    version: 2.8.5
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v13, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v14), string(out))
}
