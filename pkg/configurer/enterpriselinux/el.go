package enterpriselinux

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	log "github.com/sirupsen/logrus"
)

// Configurer is the EL family specific implementation of a host configurer
type Configurer struct {
	configurer.LinuxConfigurer
}

// InstallBasePackages install all the needed base packages on the host
func (c *Configurer) InstallBasePackages() error {
	err := c.FixContainerizedHost()
	if err != nil {
		return err
	}

	return c.Host.Exec("sudo yum install -y curl socat iptables iputils gzip")
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
	if c.Host.Exec("sudo dmidecode -s system-manufacturer|grep -q EC2") == nil {
		if c.Host.Exec("sudo yum install -q -y rh-amazon-rhui-client") == nil {
			log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", c.Host)
		}
	}

	if c.Host.Exec("sudo yum-config-manager --enable rhel-7-server-rhui-extras-rpms && sudo yum makecache fast") == nil {
		log.Infof("%s: enabled rhel-7-server-rhui-extras-rpms repository", c.Host)
	}

	return c.LinuxConfigurer.InstallEngine(engineConfig)
}
