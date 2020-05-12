package configurer

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/host"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	Host *host.Host
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
