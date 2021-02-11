package enterpriselinux

import (
	"github.com/Mirantis/mcc/pkg/configurer"
	common "github.com/Mirantis/mcc/pkg/product/common/api"

	"github.com/k0sproject/rig/os"
	"github.com/k0sproject/rig/os/linux"
	log "github.com/sirupsen/logrus"
)

// Configurer is the EL family specific implementation of a host configurer
type Configurer struct {
	linux.EnterpriseLinux
	configurer.LinuxConfigurer
}

// InstallMKEBasePackages install all the needed base packages on the host
func (c Configurer) InstallMKEBasePackages(h os.Host) error {
	return c.InstallPackage(h, "curl", "socat", "iptables", "iputils", "gzip")
}

// UninstallMCR uninstalls docker-ee engine
func (c Configurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	err := h.Exec("sudo docker system prune -f")
	if err != nil {
		return err
	}
	if err := c.StopService(h, "docker"); err != nil {
		return err
	}
	if err := c.StopService(h, "containerd"); err != nil {
		return err
	}
	return h.Exec("sudo yum remove -y docker-ee docker-ee-cli")
}

// InstallMCR install Docker EE engine on Linux
func (c Configurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	if h.Exec("sudo dmidecode -s system-manufacturer|grep -q EC2") == nil {
		if c.InstallPackage(h, "rh-amazon-rhui-client") == nil {
			log.Infof("%s: appears to be an AWS EC2 instance, installed rh-amazon-rhui-client", h)
		}
	}

	if h.Exec("sudo yum-config-manager --enable rhel-7-server-rhui-extras-rpms && sudo yum makecache fast") == nil {
		log.Infof("%s: enabled rhel-7-server-rhui-extras-rpms repository", h)
	}

	return c.LinuxConfigurer.InstallMCR(h, scriptPath, engineConfig)
}
