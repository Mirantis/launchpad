package configurer

import (
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/constant"
	log "github.com/sirupsen/logrus"
)

// WindowsConfigurer is a generic windows host configurer
type WindowsConfigurer struct {
	Host *api.Host
}

// InstallEngine install Docker EE engine on Windows
func (c *WindowsConfigurer) InstallEngine(engineConfig *api.EngineConfig) error {
	if len(c.Host.DaemonConfig) > 0 {
		daemonJSONData, err := json.Marshal(c.Host.DaemonConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal daemon json config: %w", err)
		}
		log.Debugf(`writing C:\ProgramData\Docker\config\daemon.json`)
		err = c.WriteFile(`C:\ProgramData\Docker\config\daemon.json`, string(daemonJSONData), "0700")
		if err != nil {
			return err
		}
	}
	if c.Host.Metadata.EngineVersion == engineConfig.Version {
		return nil
	}
	scriptURL := fmt.Sprintf("%sinstall.ps1", constant.EngineInstallURL)
	dlCommand := fmt.Sprintf("$ProgressPreference = 'SilentlyContinue'; [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest %s -UseBasicParsing -OutFile install.ps1", scriptURL)
	err := c.Host.ExecCmd("powershell", dlCommand, false, false)
	if err != nil {
		return err
	}
	installCommand := fmt.Sprintf("set DOWNLOAD_URL=%s && set DOCKER_VERSION=%s && set CHANNEL=%s && powershell -ExecutionPolicy Bypass -File install.ps1 -Verbose", engineConfig.RepoURL, engineConfig.Version, engineConfig.Channel)
	err = c.Host.Exec(installCommand)
	if err != nil {
		return err
	}

	return nil
}

// UninstallEngine uninstalls docker-ee engine
// TODO: actually uninstall, the install.ps1 script has '-Uninstall' option for this.
// There's also some uninstall intructions on MS site: https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-docker/configure-docker-daemon#uninstall-docker
func (c *WindowsConfigurer) UninstallEngine(engineConfig *api.EngineConfig) error {
	return c.Host.Exec("docker system prune -f")
}

// RestartEngine restarts Docker EE engine
func (c *WindowsConfigurer) RestartEngine() error {
	// TODO: handle restart
	return nil
}

// ResolveHostname resolves hostname
func (c *WindowsConfigurer) ResolveHostname() string {
	output, err := c.Host.ExecWithOutput("powershell $env:COMPUTERNAME")
	if err != nil {
		return "localhost"
	}
	return strings.TrimSpace(output)
}

// ResolveInternalIP resolves internal ip from private interface
func (c *WindowsConfigurer) ResolveInternalIP() (string, error) {
	output, err := c.Host.ExecWithOutput(fmt.Sprintf(`powershell -Command "(Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias \"%s\").IPAddress"`, c.Host.PrivateInterface))
	if err != nil {
		return c.Host.Address, err
	}
	return strings.TrimSpace(output), nil
}

// IsContainerized checks if host is actually a container
func (c *WindowsConfigurer) IsContainerized() bool {
	return false
}

// SELinuxEnabled is SELinux enabled
func (c *WindowsConfigurer) SELinuxEnabled() bool {
	return false
}

// DockerCommandf accepts a printf-like template string and arguments
// and builds a command string for running the docker cli on the host
func (c *WindowsConfigurer) DockerCommandf(template string, args ...interface{}) string {
	return fmt.Sprintf("docker.exe %s", fmt.Sprintf(template, args...))
}

// ValidateFacts validates all the collected facts so we're sure we can proceed with the installation
func (c *WindowsConfigurer) ValidateFacts() error {
	// TODO How to validate private address to be node local address?
	return nil
}

// WriteFile writes file to host with given contents
func (c *WindowsConfigurer) WriteFile(path string, data string, permissions string) error {
	// TODO Once we know how to handle permissions, change this to use similar process with linux:
	// create tmp; pipe stdin to tmp; mv tmp --> real; apply permissions; rm tmp
	err := c.Host.ExecCmd(fmt.Sprintf(`powershell -Command "$Input | New-Item -Path %s -Force -ItemType File"`, path), data, false, true)
	if err != nil {
		return err
	}
	return nil
}
