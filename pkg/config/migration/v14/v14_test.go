package v13

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestV15Migration(t *testing.T) {
	v14 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.4
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
	v15 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.5
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
    version: 3.1.1
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v14, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v15), string(out))
}

func TestV15MigrationNoOverwrite(t *testing.T) {
	v14 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.4
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
    version: 3.7.4
  msr:
    version: 3.1.1
`)
	// go's YAML marshal does not add the --- header
	v15 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.5
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
    version: 3.7.4
  msr:
    version: 3.1.1
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v14, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v15), string(out))
}
