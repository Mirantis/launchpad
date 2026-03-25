package v15

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestVersionMigrationSimple(t *testing.T) {
	v15 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    repoURL: https://repos.mirantis.com
    version: 25.0.14
  mke:
    adminUsername: admin
    imageRepo: docker.io/mirantis
    version: 3.8.6
`)

	v16 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mcr:
    channel: stable-25.0.14
    repoURL: https://repos.mirantis.com
  mke:
    adminUsername: admin
    imageRepo: docker.io/mirantis
    version: 3.8.6
`)
	in := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(v15, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v16), string(out))
}

func TestVersionMigrationDefault(t *testing.T) {
	v15 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    repoURL: https://repos.mirantis.com
    version: 25.0
  mke:
    adminUsername: admin
    imageRepo: docker.io/mirantis
    version: 3.8.6
`)

	v16 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mcr:
    channel: stable-25.0
    repoURL: https://repos.mirantis.com
  mke:
    adminUsername: admin
    imageRepo: docker.io/mirantis
    version: 3.8.6
`)
	in := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(v15, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v16), string(out))
}

func TestVersionMigrationSlim(t *testing.T) {
	v15 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    repoURL: https://repos.mirantis.com
    version: 25
  mke:
    adminUsername: admin
    imageRepo: docker.io/mirantis
    version: 3.8.6
`)

	v16 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mcr:
    channel: stable-25.0
    repoURL: https://repos.mirantis.com
  mke:
    adminUsername: admin
    imageRepo: docker.io/mirantis
    version: 3.8.6
`)
	in := make(map[string]any)
	require.NoError(t, yaml.Unmarshal(v15, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v16), string(out))
}
