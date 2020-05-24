package enterpriselinux

import (
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/configurer"
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
	return c.Host.Exec("sudo yum install -y curl")
}

// InstallEngine install Docker EE engine on Linux
func (c *Configurer) InstallEngine(engineConfig *api.EngineConfig) error {
	if c.Host.EngineConfigRef != "" {
		engineHostconfig := engineConfig.GetDaemonConfig(c.Host.EngineConfigRef)
		if engineHostconfig == nil {
			return fmt.Errorf("could not find engine config with name %s", c.Host.EngineConfigRef)
		}
		if c.SELinuxEnabled() {
			engineHostconfig.Config["selinux-enabled"] = true
		}
	}
	// FIXME How to inject "selinux-enabled":true if there's no config reference!?!?!

	return c.LinuxConfigurer.InstallEngine(engineConfig)
}

// SELinuxEnabled is SELinux enabled
func (c *Configurer) SELinuxEnabled() bool {
	output, err := c.Host.ExecWithOutput("sudo getenforce")
	if err != nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(output)) == "enforcing"
}
