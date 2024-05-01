package v14

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestVersionMigration(t *testing.T) {
	v14 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.4
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
    swarmInstallFlags:
    - --autolock
    - --cert-expiry 60h0m0s
    swarmUpdateCommands:
    - command1
    - command2
    upgradeFlags:
    - --force-recent-backup
    - --force-minimums
    version: 3.7.3
`)

	v15 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    installURLLinux: https://get.mirantis.com/
    installURLWindows: https://get.mirantis.com/install.ps1
    repoURL: https://repos.mirantis.com
    swarmInstallFlags:
    - --autolock
    - --cert-expiry 60h0m0s
    swarmUpdateCommands:
    - command1
    - command2
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
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v14, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v15), string(out))
}

func TestVersionMigrationNoOverwrite(t *testing.T) {
	v14 := []byte(`---
apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    installURLLinux: https://get.mirantis.com/
    installURLWindows: https://get.mirantis.com/install.ps1
    repoURL: https://repos.mirantis.com
    swarmInstallFlags:
    - --autolock
    - --cert-expiry 60h0m0s
    swarmUpdateCommands:
    - command1
    - command2
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
`)

	v15 := []byte(`apiVersion: launchpad.mirantis.com/mke/v1.5
kind: mke
spec:
  mcr:
    channel: stable
    installURLLinux: https://get.mirantis.com/
    installURLWindows: https://get.mirantis.com/install.ps1
    repoURL: https://repos.mirantis.com
    swarmInstallFlags:
    - --autolock
    - --cert-expiry 60h0m0s
    swarmUpdateCommands:
    - command1
    - command2
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
`)
	in := make(map[string]interface{})
	require.NoError(t, yaml.Unmarshal(v14, in))
	require.NoError(t, Migrate(in))
	out, err := yaml.Marshal(in)
	require.NoError(t, err)
	require.Equal(t, string(v15), string(out))
}
