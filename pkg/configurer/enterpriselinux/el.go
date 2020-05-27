package enterpriselinux

import (
	"encoding/json"
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
	daemonJSON := make(map[string]interface{})
	output, err := c.Host.ExecWithOutput("sudo ls /etc/docker/daemon.json && sudo cat /etc/docker/daemon.json")
	if err == nil {
		json.Unmarshal([]byte(output), &daemonJSON)
	}
	if c.SELinuxEnabled() {
		daemonJSON["selinux-enabled"] = true
	}

	json, err := json.Marshal(daemonJSON)
	if err != nil {
		return err
	}
	c.Host.WriteFile("/etc/docker/daemon.json", string(json), "0700")
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
