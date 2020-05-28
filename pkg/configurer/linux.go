package configurer

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/util"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	log "github.com/sirupsen/logrus"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	Host *api.Host
}

// InstallEngine install Docker EE engine on Linux
func (c *LinuxConfigurer) InstallEngine(engineConfig *api.EngineConfig) error {
	if c.Host.Metadata.EngineVersion == engineConfig.Version {
		return nil
	}
	cmd := fmt.Sprintf("curl %s | DOCKER_URL=%s CHANNEL=%s VERSION=%s bash", engineConfig.InstallURL, engineConfig.RepoURL, engineConfig.Channel, engineConfig.Version)
	err := c.Host.Exec(cmd)
	if err != nil {
		return err
	}

	err = c.Host.Exec("sudo systemctl enable --now docker")
	if err != nil {
		return err
	}
	return nil
}

// RestartEngine restarts Docker EE engine
func (c *LinuxConfigurer) RestartEngine() error {
	return c.Host.Exec("sudo systemctl restart docker")
}

// ResolveHostname resolves hostname
func (c *LinuxConfigurer) ResolveHostname() string {
	hostname, _ := c.Host.ExecWithOutput("hostname -s")

	return hostname
}

// ResolveInternalIP resolves internal ip from private interface
func (c *LinuxConfigurer) ResolveInternalIP() (string, error) {
	output, err := c.Host.ExecWithOutput(fmt.Sprintf("ip -o addr show dev %s scope global", c.Host.PrivateInterface))
	if err != nil {
		return "", fmt.Errorf("failed to find private interface with name %s: %s", c.Host.PrivateInterface, output)
	}
	lines := strings.Split(output, "\r\n")
	for _, line := range lines {
		items := strings.Fields(line)
		if len(items) < 4 {
			log.Debugf("not enough items in ip address line (%s), skipping...", items)
			continue
		}
		addrItems := strings.Split(items[3], "/")
		if addrItems[0] != c.Host.Address {
			if util.IsValidAddress(addrItems[0]) {
				return addrItems[0], nil
			}

			return "", fmt.Errorf("found address %s for interface %s but it does not seem to be valid address", addrItems[0], c.Host.PrivateInterface)
		}
	}
	// FIXME If we get this far should we just bail out with error!?!?
	return c.Host.Address, nil
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

// ValidateFacts validates all the collected facts so we're sure we can proceed with the installation
func (c *LinuxConfigurer) ValidateFacts() error {
	localAddresses, err := c.getHostLocalAddresses()
	if err != nil {
		return fmt.Errorf("failed to find host local addresses: %w", err)
	}

	if !util.StringSliceContains(localAddresses, c.Host.Metadata.InternalAddress) {
		return fmt.Errorf("discovered private address %s does not seem to be a node local address (%s). Make sure you've set correct 'privateInterface' for the host in config", c.Host.Metadata.InternalAddress, strings.Join(localAddresses, ","))
	}

	return nil
}

func (c *LinuxConfigurer) getHostLocalAddresses() ([]string, error) {
	output, err := c.Host.ExecWithOutput("sudo hostname --all-ip-addresses")
	if err != nil {
		return nil, err
	}

	return strings.Split(output, " "), nil
}
