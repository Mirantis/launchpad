package centos

import (
	"github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/registry"
)

// Configurer is the CentOS specific implementation of a host configurer
type Configurer struct {
	enterpriselinux.Configurer
}

// InstallMKEBasePackages install all the needed base packages on the host
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	return h.Exec("sudo yum install -y curl socat iptables iputils gzip")
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "centos"
		},
		func() interface{} {
			return Configurer{}
		},
	)
}
