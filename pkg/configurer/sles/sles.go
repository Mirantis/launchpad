package sles

import (
	"strings"

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

// InstallMKEBasePackages installs the needed base packages on Ubuntu.
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	return c.InstallPackage(h, "curl", "socat")
}

// UninstallMCR uninstalls docker-ee engine.
func (c Configurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	err := h.Exec("docker system prune -f")
	if err != nil {
		return err
	}

	if err := c.StopService(h, "docker"); err != nil {
		return err
	}

	if err := c.StopService(h, "containerd"); err != nil {
		return err
	}

	return h.Exec("sudo zypper -n remove -y --clean-deps docker-ee docker-ee-cli")
}

// LocalAddresses returns a list of local addresses, SLES12 has an old version of "hostname" without "--all-ip-addresses" and because of that, ip addr show is used here.
func (c Configurer) LocalAddresses(h os.Host) ([]string, error) {
	output, err := h.ExecOutput("ip addr show | grep 'inet ' | awk '{print $2}' | cut -d/ -f1")
	if err != nil {
		return nil, err
	}

	return strings.Fields(output), nil
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
