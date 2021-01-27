package configurer

import (
	"bufio"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/exec"
	ps "github.com/Mirantis/mcc/pkg/powershell"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"

	"github.com/hashicorp/go-version"
)

// WindowsConfigurer is a generic windows host configurer
type WindowsConfigurer struct {
	Host Host

	PowerShellVersion *version.Version
}

// Pwd returns the current working directory
func (c *WindowsConfigurer) Pwd() string {
	pwd, err := c.Host.ExecWithOutput("echo %cd%")
	if err != nil {
		return ""
	}
	return pwd
}

// MCRConfigPath returns the configuration file path
func (c *WindowsConfigurer) MCRConfigPath() string {
	return `C:\ProgramData\Docker\config\daemon.json`
}

type rebootable interface {
	Reboot() error
}

// InstallMCR install MCR on Windows
func (c *WindowsConfigurer) InstallMCR(scriptPath string, engineConfig common.MCRConfig) error {
	pwd, err := c.Host.ExecWithOutput("echo %cd%")
	if err != nil {
		return err
	}
	base := path.Base(scriptPath)
	installer := pwd + "\\" + base + ".ps1"
	err = c.Host.WriteFileLarge(scriptPath, installer)
	if err != nil {
		return err
	}

	defer c.DeleteFile(installer)

	installCommand := fmt.Sprintf("set DOWNLOAD_URL=%s && set DOCKER_VERSION=%s && set CHANNEL=%s && powershell -ExecutionPolicy Bypass -NoProfile -NonInteractive -File %s -Verbose", engineConfig.RepoURL, engineConfig.Version, engineConfig.Channel, ps.DoubleQuote(installer))

	log.Infof("%s: running installer", c.Host)

	output, err := c.Host.ExecWithOutput(installCommand)
	if err != nil {
		return err
	}

	if strings.Contains(output, "Your machine needs to be rebooted") {
		log.Warnf("%s: host needs to be rebooted", c.Host)
		if rh, ok := c.Host.(rebootable); ok {
			return rh.Reboot()
		}
		return fmt.Errorf("%s: host can't be rebooted", c.Host)
	}

	return nil
}

// UninstallMCR uninstalls docker-ee engine
// This relies on using the http://get.mirantis.com/install.ps1 script with the '-Uninstall' option, and some cleanup as per
// https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-docker/configure-docker-daemon#how-to-uninstall-docker
func (c *WindowsConfigurer) UninstallMCR(scriptPath string, engineConfig common.MCRConfig) error {
	err := c.Host.Exec(c.DockerCommandf("system prune --volumes --all -f"))
	if err != nil {
		return err
	}

	pwd := c.Pwd()
	base := path.Base(scriptPath)
	uninstaller := pwd + "\\" + base + ".ps1"
	err = c.Host.WriteFileLarge(scriptPath, uninstaller)
	if err != nil {
		return err
	}
	defer c.DeleteFile(uninstaller)

	uninstallCommand := fmt.Sprintf("powershell -NonInteractive -NoProfile -ExecutionPolicy Bypass -File %s -Uninstall -Verbose", ps.DoubleQuote(uninstaller))
	return c.Host.Exec(uninstallCommand)
}

// RestartMCR restarts Docker EE engine
func (c *WindowsConfigurer) RestartMCR() error {
	c.Host.Exec("net stop com.docker.service")
	c.Host.Exec("net start com.docker.service")
	return retry.Do(
		func() error {
			return c.Host.Exec(c.DockerCommandf("ps"))
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(10),
	)
}

// ResolveHostname resolves hostname
func (c *WindowsConfigurer) ResolveHostname() string {
	output, err := c.Host.ExecWithOutput(ps.Cmd("$env:COMPUTERNAME"))
	if err != nil {
		return "localhost"
	}
	return strings.TrimSpace(output)
}

// ResolveLongHostname resolves the FQDN (long) hostname
func (c *WindowsConfigurer) ResolveLongHostname() string {
	output, err := c.Host.ExecWithOutput(ps.Cmd("([System.Net.Dns]::GetHostByName(($env:COMPUTERNAME))).Hostname"))
	if err != nil {
		return "localhost"
	}
	return strings.TrimSpace(output)
}

// ResolveInternalIP resolves internal ip from private interface
func (c *WindowsConfigurer) ResolveInternalIP(privateInterface, publicIP string) (string, error) {
	output, err := c.interfaceIP(privateInterface)
	if err != nil {
		if !strings.HasPrefix(privateInterface, "vEthernet") {
			ve := fmt.Sprintf("vEthernet (%s)", privateInterface)
			log.Tracef("%s: trying %s as a private interface alias", c.Host, ve)
			return c.interfaceIP(ve)
		}

		return "", err
	}
	addr := strings.TrimSpace(output)
	if addr != publicIP {
		if util.IsValidAddress(addr) {
			log.Infof("%s: using %s as private IP", c.Host, addr)
			return addr, nil
		}
	}

	log.Infof("%s: using %s as private IP", c.Host, publicIP)

	return publicIP, nil
}

func (c *WindowsConfigurer) interfaceIP(iface string) (string, error) {
	output, err := c.Host.ExecWithOutput(ps.Cmd(fmt.Sprintf(`(Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias %s).IPAddress`, ps.SingleQuote(iface))))
	if err != nil {
		return "", err
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

// ValidateLocalhost returns an error if "localhost" is not local on the host
func (c *WindowsConfigurer) ValidateLocalhost() error {
	err := c.Host.Exec(ps.Cmd(fmt.Sprintf(`"$ips=[System.Net.Dns]::GetHostAddresses('localhost'); Get-NetIPAddress -IPAddress $ips"`)))
	if err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host")
	}
	return nil
}

// LocalAddresses returns a list of local addresses
func (c *WindowsConfigurer) LocalAddresses() ([]string, error) {
	output, err := c.Host.ExecWithOutput(ps.Cmd(`(Get-NetIPAddress).IPV4Address`))
	if err != nil {
		return nil, err
	}
	var lines []string
	// bufio used to split lines on windows
	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, nil
}

// CheckPrivilege returns an error if the user does not have admin access to the host
func (c *WindowsConfigurer) CheckPrivilege() error {
	privCheck := "\"$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent()); if (!$currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { $host.SetShouldExit(1) }\""

	if c.Host.Exec(ps.Cmd(privCheck)) != nil {
		return fmt.Errorf("user does not have administrator rights on the host")
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host
func (c *WindowsConfigurer) AuthenticateDocker(user, pass, imageRepo string) error {
	// the --pasword-stdin seems to hang in windows
	return c.Host.Exec(c.DockerCommandf("login -u %s -p %s %s", user, pass, imageRepo), exec.RedactString(user, pass), exec.AllowWinStderr())
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

	tempFile, err := c.Host.ExecWithOutput(ps.Cmd("New-TemporaryFile | Write-Host"))
	if err != nil {
		return err
	}
	defer c.Host.ExecWithOutput(fmt.Sprintf("del \"%s\"", tempFile))

	err = c.Host.Exec(ps.Cmd(fmt.Sprintf(`$Input | Out-File -FilePath %s`, ps.SingleQuote(tempFile))), exec.Stdin(data))
	if err != nil {
		return err
	}

	err = c.Host.Exec(ps.Cmd(fmt.Sprintf(`Move-Item -Force -Path %s -Destination %s`, ps.SingleQuote(tempFile), ps.SingleQuote(path))))
	if err != nil {
		return err
	}

	return nil
}

// ReadFile reads a files contents from the host.
func (c *WindowsConfigurer) ReadFile(path string) (string, error) {
	return c.Host.ExecWithOutput(fmt.Sprintf(`type %s`, ps.DoubleQuote(path)))
}

// DeleteFile deletes a file from the host.
func (c *WindowsConfigurer) DeleteFile(path string) error {
	return c.Host.Exec(fmt.Sprintf(`del /f %s`, ps.DoubleQuote(path)))
}

// FileExist checks if a file exists on the host
func (c *WindowsConfigurer) FileExist(path string) bool {
	return c.Host.Exec(ps.Cmd(fmt.Sprintf(`if (!(Test-Path -Path \"%s\")) { exit 1 }`, path))) == nil
}

// UpdateEnvironment updates the hosts's environment variables
func (c *WindowsConfigurer) UpdateEnvironment(env map[string]string) error {
	for k, v := range env {
		err := c.Host.Exec(fmt.Sprintf(`setx %s %s`, ps.DoubleQuote(k), ps.DoubleQuote(v)))
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupEnvironment removes environment variable configuration
func (c *WindowsConfigurer) CleanupEnvironment(env map[string]string) error {
	for k := range env {
		c.Host.Exec(ps.Cmd(fmt.Sprintf(`[Environment]::SetEnvironmentVariable(%s, $null, 'User')`, ps.SingleQuote(k))))
		c.Host.Exec(ps.Cmd(fmt.Sprintf(`[Environment]::SetEnvironmentVariable(%s, $null, 'Machine')`, ps.SingleQuote(k))))
	}
	return nil
}

// ResolvePrivateInterface tries to find a private network interface
func (c *WindowsConfigurer) ResolvePrivateInterface() (string, error) {
	output, err := c.Host.ExecWithOutput(ps.Cmd(`(Get-NetConnectionProfile -NetworkCategory Private | Select-Object -First 1).InterfaceAlias`))
	if err != nil || output == "" {
		output, err = c.Host.ExecWithOutput(ps.Cmd(`(Get-NetConnectionProfile | Select-Object -First 1).InterfaceAlias`))
	}
	if err != nil || output == "" {
		return "", fmt.Errorf("failed to detect a private network interface, define the host privateInterface manually")
	}
	return strings.TrimSpace(output), nil
}

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error
func (c *WindowsConfigurer) HTTPStatus(url string) (int, error) {
	log.Debugf("%s: requesting %s", c.Host, url)
	output, err := c.Host.ExecWithOutput(ps.Cmd(fmt.Sprintf(`[int][System.Net.WebRequest]::Create(%s).GetResponse().StatusCode`, ps.SingleQuote(url))))
	if err != nil {
		return -1, err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("invalid response: %s", err.Error())
	}
	return status, nil
}

// JoinPath joins a path
func (c *WindowsConfigurer) JoinPath(parts ...string) string {
	return strings.Join(parts, "\\")
}

// RebootCommand returns a command string that will reboot the host
func (c *WindowsConfigurer) RebootCommand() string {
	return "shutdown /r /t 5"
}
