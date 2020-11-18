package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestVariableMigration(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1"
kind: DockerEnterprise
spec:
  ucp:
    version: $MKE_VERSION
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.1
kind: mke
spec:
  mke:
    version: $$MKE_VERSION
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}

func TestReplicaConfigDeprecation(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1"
kind: DockerEnterprise
spec:
  dtr:
    replicaConfig: sequential
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.1
kind: mke+msr
spec:
  msr: {}
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}

func TestCredentialsMigration(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1"
kind: DockerEnterprise
spec:
  ucp:
    installFlags:
      - --admin-username "foo"
      - --test
      - --admin-password="barbar"
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.1
kind: mke
spec:
  mke:
    adminPassword: barbar
    adminUsername: foo
    installFlags:
    - --test
`)
	// looks like yaml.Marshal alphabetically sorts these, no matter which way the code is flipped.

	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}

func TestCredentialsMigrationMSRnoMKE(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1"
kind: DockerEnterprise
spec:
  dtr:
    installFlags:
      - --ucp-username "foo"
      - --test
      - --ucp-password="barbar"
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.1
kind: mke+msr
spec:
  mke:
    adminPassword: barbar
    adminUsername: foo
  msr:
    installFlags:
    - --test
`)

	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}

func TestCredentialsMigrationMSRandMKE(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1"
kind: DockerEnterprise
spec:
  ucp:
    installFlags:
      - --admin-username foo
      - --admin-password barbar
  dtr:
    installFlags:
      - --ucp-username "food"
      - --test
      - --ucp-password="bardbard"
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.1
kind: mke+msr
spec:
  mke:
    adminPassword: barbar
    adminUsername: foo
    installFlags: []
  msr:
    installFlags:
    - --ucp-username "food"
    - --test
    - --ucp-password="bardbard"
`)

	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}

func TestRoleMigration(t *testing.T) {
	v1 := []byte(`---
apiVersion: "launchpad.mirantis.com/v1"
kind: DockerEnterprise
spec:
  hosts:
    - address: 10.0.0.1
      role: manager
    - address: 10.0.0.2
      role: dtr
`)
	// go's YAML marshal does not add the --- header
	v11 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.1
kind: mke+msr
spec:
  hosts:
  - address: 10.0.0.1
    role: manager
  - address: 10.0.0.2
    role: msr
`)
	// looks like yaml.Marshal alphabetically sorts these, no matter which way the code is flipped.

	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v1, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v11), string(out))
}
