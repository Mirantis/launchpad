package enterpriselinux

import (
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"
)

var (
	RhelMCRRepoTemplate = `
[mirantis-mcr]
name=Docker EE Stable - $basearch
baseurl=${REPO}/${OS}/$releasever/$basearch/${CHANNEL}
enabled=1
gpgcheck=1
gpgkey=${REPO}/${OS}/gpg
module_hotfixes=1
`
)

// Rhel RedHat Enterprise Linux.
type Rhel struct {
	Configurer
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "rhel"
		},
		func() interface{} {
			return Rhel{}
		},
	)
}
