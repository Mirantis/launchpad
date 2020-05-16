package configurer

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	Host *config.Host
}

// InstallEngine install Docker EE engine on Linux
func (c *LinuxConfigurer) InstallEngine(engineConfig *config.EngineConfig) error {
	cmd := fmt.Sprintf("curl %s | DOCKER_URL=%s CHANNEL=%s VERSION=%s bash", engineConfig.InstallURL, engineConfig.RepoURL, engineConfig.Channel, engineConfig.Version)
	err := c.Host.Exec(cmd)
	if err != nil {
		return err
	}

	err = c.Host.Exec("sudo systemctl enable --now docker")
	if err != nil {
		return err
	}

	log.Infof("Succesfully installed engine (%s) on %s", engineConfig.Version, c.Host.Address)
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

// IsContainerized checks if host is actually a container
func (c *LinuxConfigurer) IsContainerized() bool {
	err := c.Host.Exec("grep 'container=docker' /proc/1/environ")
	if err != nil {
		return false
	}
	return true
}

// FixContainerizedHost configures host if host is containerized environment
func (c *LinuxConfigurer) FixContainerizedHost() error {
	if c.IsContainerized() {
		return c.Host.Exec("sudo mount --make-rshared /")
	}
	return nil
}

// DockerCommandf accepts a printf-like template string and arguments
// and builds a command string for running the docker cli on the host
func (c *LinuxConfigurer) DockerCommandf(template string, args ...interface{}) string {
	return fmt.Sprintf("sudo docker %s", fmt.Sprintf(template, args...))
}
