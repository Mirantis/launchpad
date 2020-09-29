package configurer

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/exec"
	log "github.com/sirupsen/logrus"

	escape "github.com/alessio/shellescape"
	"github.com/hashicorp/go-version"
)

// WindowsConfigurer is a generic windows host configurer
type WindowsConfigurer struct {
	Host *api.Host

	PowerShellVersion *version.Version
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

	pwd, err := c.Host.ExecWithOutput("echo %cd%")
	if err != nil {
		return err
	}
	base := path.Base(c.Host.Metadata.EngineInstallScript)
	installer := pwd + "\\" + base
	err = c.Host.Connection.WriteFileLarge(c.Host.Metadata.EngineInstallScript, pwd)
	if err != nil {
		log.Errorf("failed: %s", err.Error())
		return err
	}

	defer c.Host.Exec(fmt.Sprintf("del %s", installer))

	installCommand := fmt.Sprintf("set DOWNLOAD_URL=%s && set DOCKER_VERSION=%s && set CHANNEL=%s && powershell -ExecutionPolicy Bypass -File %s -Verbose", engineConfig.RepoURL, engineConfig.Version, engineConfig.Channel, installer)
	return c.Host.Exec(installCommand)
}

// UninstallEngine uninstalls docker-ee engine
// This relies on using the http://get.mirantis.com/install.ps1 script with the '-Uninstall' option, and some cleanup as per
// https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-docker/configure-docker-daemon#how-to-uninstall-docker
func (c *WindowsConfigurer) UninstallEngine(engineConfig *api.EngineConfig) error {
	err := c.Host.Exec(c.DockerCommandf("system prune --volumes --all -f"))
	if err != nil {
		return err
	}

	pwd, err := c.Host.ExecWithOutput("echo %cd%")
	if err != nil {
		return err
	}
	base := path.Base(c.Host.Metadata.EngineInstallScript)
	uninstaller := pwd + "\\" + base
	c.Host.Connection.WriteFileLarge(uninstaller, c.Host.Metadata.EngineInstallScript)

	defer c.Host.Exec(fmt.Sprintf("del %s", uninstaller))

	uninstallCommand := fmt.Sprintf("powershell -ExecutionPolicy Bypass -File %s -Uninstall -Verbose", uninstaller)
	return c.Host.Exec(uninstallCommand)
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

// ResolveLongHostname resolves the FQDN (long) hostname
func (c *WindowsConfigurer) ResolveLongHostname() string {
	output, err := c.Host.ExecWithOutput("powershell ([System.Net.Dns]::GetHostByName(($env:COMPUTERNAME))).Hostname")
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
	if !c.validateLocal("localhost") {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host")
	}

	if !c.validateLocal(c.Host.Metadata.InternalAddress) {
		return fmt.Errorf("discovered private address %s does not seem to be a node local address. Make sure you've set correct 'privateInterface' for the host in config", c.Host.Metadata.InternalAddress)
	}

	return nil
}

func (c *WindowsConfigurer) validateLocal(address string) bool {
	err := c.Host.Exec(fmt.Sprintf(`powershell "$ips=[System.Net.Dns]::GetHostAddresses(\"%s\"); Get-NetIPAddress -IPAddress $ips"`, address))
	return err == nil
}

// CheckPrivilege returns an error if the user does not have admin access to the host
func (c *WindowsConfigurer) CheckPrivilege() error {
	privCheck := "$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent()); if ($currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { Write-Host 'User has admin privileges'; exit 0; } else { Write-Host 'User does not have admin privileges'; exit 1 }"

	if c.Host.Exec("powershell.exe", exec.Stdin(privCheck)) != nil {
		return fmt.Errorf("user does not have administrator rights on the host")
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host
func (c *WindowsConfigurer) AuthenticateDocker(user, pass, imageRepo string) error {
	// the --pasword-stdin seems to hang in windows
	return c.Host.Exec(c.DockerCommandf("login -u %s -p %s %s", user, pass, imageRepo), exec.Redact(fmt.Sprintf("(?:%s|%s)", regexp.QuoteMeta(user), regexp.QuoteMeta(pass))))
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

	err = c.Host.Exec(fmt.Sprintf(`powershell -Command "$Input | Out-File -FilePath \"%s\""`, tempFile), exec.Stdin(data))
	if err != nil {
		return err
	}

	err = c.Host.Exec(fmt.Sprintf(`powershell -Command "Move-Item -Force -Path \"%s\" -Destination \"%s\""`, tempFile, path))
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
	return c.Host.Exec(fmt.Sprintf(`del /f "%s"`, path))
}

// FileExist checks if a file exists on the host
func (c *WindowsConfigurer) FileExist(path string) bool {
	return c.Host.Exec(fmt.Sprintf(`powershell -Command "if (!(Test-Path -Path \"%s\")) { exit 1 }"`, path)) == nil
}

// UpdateEnvironment updates the hosts's environment variables
func (c *WindowsConfigurer) UpdateEnvironment() error {
	for k, v := range c.Host.Environment {
		err := c.Host.Exec(fmt.Sprintf(`setx %s %s`, escape.Quote(k), escape.Quote(v)))
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupEnvironment removes environment variable configuration
func (c *WindowsConfigurer) CleanupEnvironment() error {
	for k := range c.Host.Environment {
		c.Host.Exec(fmt.Sprintf(`powershell "[Environment]::SetEnvironmentVariable('%s', $null, 'User')"`, escape.Quote(k)))
		c.Host.Exec(fmt.Sprintf(`powershell "[Environment]::SetEnvironmentVariable('%s', $null, 'Machine')"`, escape.Quote(k)))
	}
	return nil
}

// ResolvePrivateInterface tries to find a private network interface
func (c *WindowsConfigurer) ResolvePrivateInterface() (string, error) {
	output, err := c.Host.ExecWithOutput(`powershell "(Get-NetConnectionProfile -NetworkCategory Private | Select-Object -First 1).InterfaceAlias"`)
	if err != nil || output == "" {
		output, err = c.Host.ExecWithOutput(`powershell "(Get-NetConnectionProfile | Select-Object -First 1).InterfaceAlias"`)
	}
	if err != nil || output == "" {
		return "", fmt.Errorf("failed to detect a private network interface, define the host privateInterface manually")
	}
	return strings.TrimSpace(output), nil
}

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error
func (c *WindowsConfigurer) HTTPStatus(url string) (int, error) {
	log.Debugf("%s: requesting %s", c.Host.Address, url)
	output, err := c.Host.ExecWithOutput(fmt.Sprintf(`powershell "[int][System.Net.WebRequest]::Create('%s').GetResponse().StatusCode"`, url))
	if err != nil {
		return -1, err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("%s: invalid response: %s", c.Host.Address, err.Error())
	}
	return status, nil
}
