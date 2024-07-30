package v15

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestVersionMigration(t *testing.T) {
	v15 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    installURLLinux: https://get.mirantis.com/
    installURLWindows: https://get.mirantis.com/install.ps1
    repoURL: https://repos.mirantis.com
    version: 23.0.8
  mke:
    adminPassword: miradmin
    adminUsername: admin
    imageRepo: docker.io/mirantis
    installFlags:
    - --san=pgedaray-mke-lb-1477b84d031720d6.elb.us-west-1.amazonaws.com
    - --nodeport-range=32768-35535
    upgradeFlags:
    - --force-recent-backup
    - --force-minimums
    version: 3.7.3
  msr:
    imageRepo: docker.io/mirantis
    installFlags:
    - --nfs-options nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2,noresvport
    - --nfs-storage-url nfs://nfs.example.com/
    replicaIDs: sequential
    version: 2.9.15
`)
	v16 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.6
kind: mke
spec:
  mcr:
    channel: stable
    installURLLinux: https://get.mirantis.com/
    installURLWindows: https://get.mirantis.com/install.ps1
    repoURL: https://repos.mirantis.com
    version: 23.0.8
  mke:
    adminPassword: miradmin
    adminUsername: admin
    imageRepo: docker.io/mirantis
    installFlags:
    - --san=pgedaray-mke-lb-1477b84d031720d6.elb.us-west-1.amazonaws.com
    - --nodeport-range=32768-35535
    upgradeFlags:
    - --force-recent-backup
    - --force-minimums
    version: 3.7.3
  msr2:
    imageRepo: docker.io/mirantis
    installFlags:
    - --nfs-options nfsvers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2,noresvport
    - --nfs-storage-url nfs://nfs.example.com/
    replicaIDs: sequential
    version: 2.9.15
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v15, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v16), string(out))
}
