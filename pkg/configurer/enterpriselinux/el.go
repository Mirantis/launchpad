package enterpriselinux

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/configurer"
)

// Configurer is the EL family specific implementation of a host configurer
type Configurer struct {
	configurer.LinuxConfigurer
}

// ResolveInternalIP resolves internal ip from private interface
func (c *Configurer) ResolveInternalIP() (string, error) {
	output, err := c.Host.ExecWithOutput(fmt.Sprintf("/usr/sbin/ip -o addr show dev %s scope global", c.Host.PrivateInterface))
	if err != nil {
		return "", fmt.Errorf("failed to find private interface with name %s: %s. Make sure you've set correct 'privateInterface' for the host in config", c.Host.PrivateInterface, output)
	}
	return c.ParseInternalIPFromIPOutput(output)
}

// InstallBasePackages install all the needed base packages on the host
func (c *Configurer) InstallBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo yum install -y curl socat")
}

// UninstallEngine uninstalls docker-ee engine
func (c *Configurer) UninstallEngine(engineConfig *api.EngineConfig) error {
	err := c.Host.Exec("sudo docker system prune -f")
	if err != nil {
		return err
	}
	err = c.Host.Exec("sudo systemctl stop docker")
	if err != nil {
		return err
	}
	err = c.Host.Exec("sudo systemctl stop containerd")
	if err != nil {
		return err
	}
	return c.Host.Exec("sudo yum remove -y docker-ee docker-ee-cli")
}

// InstallEngine install Docker EE engine on Linux
func (c *Configurer) InstallEngine(engineConfig *api.EngineConfig) error {
	if c.SELinuxEnabled() {
		c.Host.DaemonConfig["selinux-enabled"] = true
	}
	return c.LinuxConfigurer.InstallEngine(engineConfig)
}
