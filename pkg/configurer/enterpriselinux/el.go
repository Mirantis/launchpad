package enterpriselinux

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/configurer"

	log "github.com/sirupsen/logrus"
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

	return c.Host.Exec("sudo yum install -y curl socat iptables iputils")
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

	if c.Host.Exec("sudo dmidecode -s system-manufacturer|grep -q EC2") == nil {
		if c.Host.Exec("sudo yum install -q -y rh-amazon-rhui-client") == nil {
			log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", c.Host.Address)
		}
	}

	if c.Host.Exec("sudo yum-config-manager --enable rhel-7-server-rhui-extras-rpms && sudo yum makecache fast") == nil {
		log.Infof("%s: enabled rhel-7-server-rhui-extras-rpms repository", c.Host.Address)
	}

	return c.LinuxConfigurer.InstallEngine(engineConfig)
}
