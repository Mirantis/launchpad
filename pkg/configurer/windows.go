package configurer

import (
	"bufio"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/avast/retry-go"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	ps "github.com/k0sproject/rig/powershell"
	log "github.com/sirupsen/logrus"

	"github.com/hashicorp/go-version"
)

// WindowsConfigurer is a generic windows host configurer
type WindowsConfigurer struct {
	os.Windows

	PowerShellVersion *version.Version
}

// MCRConfigPath returns the configuration file path
func (c WindowsConfigurer) MCRConfigPath() string {
	return `C:\ProgramData\Docker\config\daemon.json`
}

type rebootable interface {
	Reboot() error
}

// InstallMCR install MCR on Windows
func (c WindowsConfigurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	pwd := c.Pwd(h)
	base := path.Base(scriptPath)
	installer := pwd + "\\" + base + ".ps1"
	err := h.Upload(scriptPath, installer)
	if err != nil {
		return err
	}

	defer c.DeleteFile(h, installer)

	installCommand := fmt.Sprintf("set DOWNLOAD_URL=%s && set DOCKER_VERSION=%s && set CHANNEL=%s && powershell -ExecutionPolicy Bypass -NoProfile -NonInteractive -File %s -Verbose", engineConfig.RepoURL, engineConfig.Version, engineConfig.Channel, ps.DoubleQuote(installer))

	log.Infof("%s: running installer", h)

	output, err := h.ExecOutput(installCommand)
	if err != nil {
		return err
	}

	if strings.Contains(output, "Your machine needs to be rebooted") {
		log.Warnf("%s: host needs to be rebooted", h)
		if rh, ok := h.(rebootable); ok {
			return rh.Reboot()
		}
		return fmt.Errorf("%s: host can't be rebooted", h)
	}

	return nil
}

// UninstallMCR uninstalls docker-ee engine
// This relies on using the http://get.mirantis.com/install.ps1 script with the '-Uninstall' option, and some cleanup as per
// https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-docker/configure-docker-daemon#how-to-uninstall-docker
func (c WindowsConfigurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	err := h.Exec(c.Dockerf(h, "system prune --volumes --all -f"))
	if err != nil {
		return err
	}

	pwd := c.Pwd(h)
	base := path.Base(scriptPath)
	uninstaller := pwd + "\\" + base + ".ps1"
	err = h.Upload(scriptPath, uninstaller)
	if err != nil {
		return err
	}
	defer c.DeleteFile(h, uninstaller)

	uninstallCommand := fmt.Sprintf("powershell -NonInteractive -NoProfile -ExecutionPolicy Bypass -File %s -Uninstall -Verbose", ps.DoubleQuote(uninstaller))
	return h.Exec(uninstallCommand)
}

// RestartMCR restarts Docker EE engine
func (c WindowsConfigurer) RestartMCR(h os.Host) error {
	h.Exec("net stop com.docker.service")
	h.Exec("net start com.docker.service")
	return retry.Do(
		func() error {
			return h.Exec(c.Dockerf(h, "ps"))
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(10),
	)
}

// ResolveInternalIP resolves internal ip from private interface
func (c WindowsConfigurer) ResolveInternalIP(h os.Host, privateInterface, publicIP string) (string, error) {
	output, err := c.interfaceIP(h, privateInterface)
	if err != nil {
		if !strings.HasPrefix(privateInterface, "vEthernet") {
			ve := fmt.Sprintf("vEthernet (%s)", privateInterface)
			log.Tracef("%s: trying %s as a private interface alias", h, ve)
			return c.interfaceIP(h, ve)
		}

		return "", err
	}
	addr := strings.TrimSpace(output)
	if addr != publicIP {
		if util.IsValidAddress(addr) {
			log.Infof("%s: using %s as private IP", h, addr)
			return addr, nil
		}
	}

	log.Infof("%s: using %s as private IP", h, publicIP)

	return publicIP, nil
}

func (c WindowsConfigurer) interfaceIP(h os.Host, iface string) (string, error) {
	output, err := h.ExecOutput(ps.Cmd(fmt.Sprintf(`(Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias %s).IPAddress`, ps.SingleQuote(iface))))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// Dockerf accepts a printf-like template string and arguments
// and builds a command string for running the docker cli on the host
func (c WindowsConfigurer) Dockerf(h os.Host, template string, args ...interface{}) string {
	return fmt.Sprintf("docker.exe %s", fmt.Sprintf(template, args...))
}

// ValidateLocalhost returns an error if "localhost" is not local on the host
func (c WindowsConfigurer) ValidateLocalhost(h os.Host) error {
	err := h.Exec(ps.Cmd(fmt.Sprintf(`"$ips=[System.Net.Dns]::GetHostAddresses('localhost'); Get-NetIPAddress -IPAddress $ips"`)))
	if err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host")
	}
	return nil
}

// LocalAddresses returns a list of local addresses
func (c WindowsConfigurer) LocalAddresses(h os.Host) ([]string, error) {
	output, err := h.ExecOutput(ps.Cmd(`(Get-NetIPAddress).IPV4Address`))
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
func (c WindowsConfigurer) CheckPrivilege(h os.Host) error {
	privCheck := "\"$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent()); if (!$currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { $host.SetShouldExit(1) }\""

	if h.Exec(ps.Cmd(privCheck)) != nil {
		return fmt.Errorf("user does not have administrator rights on the host")
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host
func (c WindowsConfigurer) AuthenticateDocker(h os.Host, user, pass, imageRepo string) error {
	// the --pasword-stdin seems to hang in windows
	return h.Exec(c.Dockerf(h, "login -u %s -p %s %s", user, pass, imageRepo), exec.RedactString(user, pass), exec.AllowWinStderr())
}

// UpdateEnvironment updates the hosts's environment variables
func (c WindowsConfigurer) UpdateEnvironment(h os.Host, env map[string]string) error {
	for k, v := range env {
		err := h.Exec(fmt.Sprintf(`setx %s %s`, ps.DoubleQuote(k), ps.DoubleQuote(v)))
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupEnvironment removes environment variable configuration
func (c WindowsConfigurer) CleanupEnvironment(h os.Host, env map[string]string) error {
	for k := range env {
		h.Exec(ps.Cmd(fmt.Sprintf(`[Environment]::SetEnvironmentVariable(%s, $null, 'User')`, ps.SingleQuote(k))))
		h.Exec(ps.Cmd(fmt.Sprintf(`[Environment]::SetEnvironmentVariable(%s, $null, 'Machine')`, ps.SingleQuote(k))))
	}
	return nil
}

// ResolvePrivateInterface tries to find a private network interface
func (c WindowsConfigurer) ResolvePrivateInterface(h os.Host) (string, error) {
	output, err := h.ExecOutput(ps.Cmd(`(Get-NetConnectionProfile -NetworkCategory Private | Select-Object -First 1).InterfaceAlias`))
	if err != nil || output == "" {
		output, err = h.ExecOutput(ps.Cmd(`(Get-NetConnectionProfile | Select-Object -First 1).InterfaceAlias`))
	}
	if err != nil || output == "" {
		return "", fmt.Errorf("failed to detect a private network interface, define the host privateInterface manually")
	}
	return strings.TrimSpace(output), nil
}

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error
func (c WindowsConfigurer) HTTPStatus(h os.Host, url string) (int, error) {
	log.Debugf("%s: requesting %s", h, url)
	output, err := h.ExecOutput(ps.Cmd(fmt.Sprintf(`[int][System.Net.WebRequest]::Create(%s).GetResponse().StatusCode`, ps.SingleQuote(url))))
	if err != nil {
		return -1, err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("invalid response: %s", err.Error())
	}
	return status, nil
}

// AuthorizeDocker does nothing on windows
func (c WindowsConfigurer) AuthorizeDocker(h os.Host) error {
	return nil
}
