package centos

import (
	"fmt"

	github.com/Mirantis/launchpad/pkg/configurer/enterpriselinux"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/registry"
)

// Configurer is the CentOS specific implementation of a host configurer.
type Configurer struct {
	enterpriselinux.Configurer
}

// InstallMKEBasePackages install all the needed base packages on the host.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	if err := c.InstallPackage(h, "curl", "socat", "iptables", "iputils", "gzip"); err != nil {
		return fmt.Errorf("failed to install base packages: %w", err)
	}
	return nil
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
