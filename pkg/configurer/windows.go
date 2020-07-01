package configurer

import (
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
	log "github.com/sirupsen/logrus"

	escape "github.com/alessio/shellescape"
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

		cfg := `C:\ProgramData\Docker\config\daemon.json`
		if c.FileExist(cfg) {
			log.Debugf("deleting %s", cfg)
			if err := c.DeleteFile(cfg); err != nil {
				return err
			}
		}

		log.Debugf("writing %s", cfg)
		if err := c.WriteFile(cfg, string(daemonJSONData), "0700"); err != nil {
			return err
		}
	}

	if c.Host.Metadata.EngineVersion == engineConfig.Version {
		return nil
	}

	installer := "install.ps1"
	c.WriteFile(installer, *c.Host.Metadata.EngineInstallScript, "0600")

	defer c.Host.Execf("del %s", installer)

	installCommand := fmt.Sprintf("set DOWNLOAD_URL=%s && set DOCKER_VERSION=%s && set CHANNEL=%s && powershell -ExecutionPolicy Bypass -File %s -Verbose", engineConfig.RepoURL, engineConfig.Version, engineConfig.Channel, installer)
	return c.Host.Exec(installCommand)
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

// CheckPrivilege returns an error if the user does not have admin access to the host
func (c *WindowsConfigurer) CheckPrivilege() error {
	privCheck := "$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent()); if ($currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { Write-Host 'User has admin privileges'; exit 0; } else { Write-Host 'User does not have admin privileges'; exit 1 }"

	if c.Host.ExecCmd("powershell.exe", privCheck, false, false) != nil {
		return fmt.Errorf("user does not have administrator rights on the host")
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host
func (c *WindowsConfigurer) AuthenticateDocker(user, pass, imageRepo string) error {
	// the --pasword-stdin seems to hang in windows
	_, err := c.Host.ExecWithOutput(c.DockerCommandf("login -u %s -p %s %s", user, pass, imageRepo))
	return err
}

// WriteFile writes file to host with given contents. Do not use for large files.
// The permissions argument is ignored on windows.
func (c *WindowsConfigurer) WriteFile(path string, data string, permissions string) error {
	if data == "" {
		return fmt.Errorf("empty content in WriteFile to %s", path)
	}

	if path == "" {
		return fmt.Errorf("empty path in WriteFile")
	}

	tempFile, err := c.Host.ExecWithOutput("powershell -Command \"New-TemporaryFile | Write-Host\"")
	if err != nil {
		return err
	}
	defer c.Host.ExecWithOutput(fmt.Sprintf("del \"%s\"", tempFile))

	err = c.Host.ExecCmd(fmt.Sprintf(`powershell -Command "$Input | Out-File -FilePath \"%s\""`, tempFile), data, false, false)
	if err != nil {
		return err
	}

	err = c.Host.ExecCmd(fmt.Sprintf(`powershell -Command "Move-Item -Force -Path \"%s\" -Destination \"%s\""`, tempFile, path), "", false, false)
	if err != nil {
		return err
	}

	return nil
}

// ReadFile reads a files contents from the host.
func (c *WindowsConfigurer) ReadFile(path string) (string, error) {
	return c.Host.ExecWithOutput(fmt.Sprintf(`type "%s"`, path))
}

// DeleteFile deletes a file from the host.
func (c *WindowsConfigurer) DeleteFile(path string) error {
	return c.Host.ExecCmd(fmt.Sprintf(`del /f "%s"`, path), "", false, false)
}

// FileExist checks if a file exists on the host
func (c *WindowsConfigurer) FileExist(path string) bool {
	return c.Host.ExecCmd(fmt.Sprintf(`powershell -Command "if (!(Test-Path -Path \"%s\")) { exit 1 }"`, path), "", false, false) == nil
}

// UpdateEnvironment updates the hosts's environment variables
func (c *WindowsConfigurer) UpdateEnvironment() error {
	for k, v := range c.Host.Environment {
		err := c.Host.ExecCmd(fmt.Sprintf(`setx %s %s`, escape.Quote(k), escape.Quote(v)), "", false, false)
		if err != nil {
			return err
		}
	}
	return nil
}
