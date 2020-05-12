package configurer

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/host"
	log "github.com/sirupsen/logrus"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	Host *host.Host
}

// InstallEngine install Docker EE engine on Linux
func (c *LinuxConfigurer) InstallEngine() error {
	err := c.Host.Exec("curl https://s3-us-west-2.amazonaws.com/internal-docker-ee-builds/install.sh | DOCKER_URL=http://repos-internal.mirantis.com.s3.amazonaws.com CHANNEL=test bash")
	if err != nil {
		return err
	}

	err = c.Host.Exec("sudo systemctl enable --now docker")
	if err != nil {
		return err
	}

	log.Infof("Succesfully installed engine on %s", c.Host.Address)
	return nil
}

// ResolveHostname resolves hostname
func (c *LinuxConfigurer) ResolveHostname() string {
	hostname, _ := c.Host.ExecWithOutput("hostname -s")

	return hostname
}

// ResolveInternalIP resolves internal ip from private interface
func (c *LinuxConfigurer) ResolveInternalIP() string {
	output, _ := c.Host.ExecWithOutput(fmt.Sprintf("ip -o addr show dev %s scope global", c.Host.PrivateInterface))
	lines := strings.Split(output, "\r\n")
	for _, line := range lines {
		items := strings.Fields(line)
		addrItems := strings.Split(items[3], "/")
		if addrItems[0] != c.Host.Address {
			return addrItems[0]
		}
	}
	return c.Host.Address
}
