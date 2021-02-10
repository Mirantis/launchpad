package sles

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
	"github.com/k0sproject/rig/os/registry"
)

// Configurer is a generic Ubuntu level configurer implementation. Some of the configurer interface implementation
// might be on OS version specific implementation such as for Bionic.
type Configurer struct {
	linux.SLES
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages installs the needed base packages on Ubuntu
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	return h.Exec("sudo zypper -n install -y curl socat")
}

// UninstallMCR uninstalls docker-ee engine
func (c Configurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	err := h.Exec("sudo docker system prune -f")
	if err != nil {
		return err
	}
	err = h.Exec("sudo systemctl stop docker")
	if err != nil {
		return err
	}
	err = h.Exec("sudo systemctl stop containerd")
	if err != nil {
		return err
	}
	return h.Exec("sudo zypper -n remove -y --clean-deps docker-ee docker-ee-cli")
}

func init() {
	registry.RegisterOSModule(
		func(os rig.OSVersion) bool {
			return os.ID == "sles"
		},
		func() interface{} {
			return Configurer{}
		},
	)
}
