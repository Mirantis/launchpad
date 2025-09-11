package configurer

import (
	"bufio"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Mirantis/launchpad/pkg/constant"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/Mirantis/launchpad/pkg/util/iputil"
	"github.com/avast/retry-go"
	"github.com/hashicorp/go-version"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	ps "github.com/k0sproject/rig/pkg/powershell"
	log "github.com/sirupsen/logrus"
)

const (
	// WindowsDockerLicenseFile filename for the docker license file on Windows machines.
	WindowsDockerLicenseFile = "docker.lic"
)

// WindowsConfigurer is a generic windows host configurer.
type WindowsConfigurer struct {
	os.Windows

	PowerShellVersion *version.Version
	DockerConfigurer
}

// MCRConfigPath returns the configuration file path.
func (c WindowsConfigurer) MCRConfigPath() string {
	return `C:\ProgramData\Docker\config\daemon.json`
}

type rebootable interface {
	Reboot() error
}

var errRebootRequired = fmt.Errorf("reboot required")

// Install MCR License.
func (c WindowsConfigurer) InstallMCRLicense(h os.Host, lic string) error {
	// Use default docker root dir if not specified in docker info
	dockerRootDir := constant.WindowsDefaultDockerRoot

	// set the docker root dir from docker info if it exists
	if info, err := c.GetDockerInfo(h); err == nil && info != (common.DockerInfo{}) {
		dockerRootDir = info.DockerRootDir
	}

	licPath := filepath.Join(dockerRootDir, WindowsDockerLicenseFile)
	if err := c.WriteFile(h, licPath, lic, "400"); err != nil {
		return fmt.Errorf("license write; %w", err)
	}
	return nil
}

// InstallMCR install MCR on Windows.
func (c WindowsConfigurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	pwd := c.Pwd(h)
	base := path.Base(scriptPath)
	installer := pwd + "\\" + base + ".ps1"
	if err := h.Upload(scriptPath, installer, fs.FileMode(0o640)); err != nil {
		return fmt.Errorf("failed to upload MCR installer: %w", err)
	}
	defer func() {
		if err := c.DeleteFile(h, installer); err != nil {
			log.Warnf("failed to delete MCR installer: %s", err.Error())
		}
	}()

	installCommand := fmt.Sprintf("set DOWNLOAD_URL=%s && set DOCKER_VERSION=%s && set CHANNEL=%s && powershell -ExecutionPolicy Bypass -NoProfile -NonInteractive -File %s -Verbose", engineConfig.RepoURL, engineConfig.Version, engineConfig.Channel, ps.DoubleQuote(installer))

	log.Infof("%s: running installer", h)

	output, err := h.ExecOutput(installCommand)
	if err != nil {
		return fmt.Errorf("failed to run MCR installer: %w", err)
	}

	if strings.Contains(output, "Your machine needs to be rebooted") {
		log.Warnf("%s: host needs to be rebooted", h)
		if rh, ok := h.(rebootable); ok {
			if err := rh.Reboot(); err != nil {
				return fmt.Errorf("%s: failed to reboot host: %w", h, err)
			}
		}
		return fmt.Errorf("%s: %w: host isn't rebootable", h, errRebootRequired)
	}

	return nil
}

// UninstallMCR uninstalls docker-ee engine
// This relies on using the http://get.mirantis.com/install.ps1 script with the '-Uninstall' option, and some cleanup as per
// https://docs.microsoft.com/en-us/virtualization/windowscontainers/manage-docker/configure-docker-daemon#how-to-uninstall-docker
func (c WindowsConfigurer) UninstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h)
	if engineConfig.Prune {
		defer c.CleanupLingeringMCR(h, info)
	}
	if getDockerError == nil {
		if err := h.Exec(c.DockerCommandf("system prune --volumes --all -f")); err != nil {
			return fmt.Errorf("prune docker: %w", err)
		}

		pwd := c.Pwd(h)
		base := path.Base(scriptPath)
		uninstaller := pwd + "\\" + base + ".ps1"
		if err := h.Upload(scriptPath, uninstaller, fs.FileMode(0o640)); err != nil {
			return fmt.Errorf("upload MCR uninstaller: %w", err)
		}
		defer func() {
			if err := c.DeleteFile(h, uninstaller); err != nil {
				log.Warnf("failed to delete MCR uninstaller: %s", err.Error())
			}
		}()

		uninstallCommand := fmt.Sprintf("powershell -NonInteractive -NoProfile -ExecutionPolicy Bypass -File %s -Uninstall -Verbose", ps.DoubleQuote(uninstaller))
		if err := h.Exec(uninstallCommand); err != nil {
			return fmt.Errorf("run MCR uninstaller: %w", err)
		}
	}

	return nil
}

// RestartMCR restarts Docker EE engine.
func (c WindowsConfigurer) RestartMCR(h os.Host) error {
	_ = h.Exec("net stop com.docker.service")
	_ = h.Exec("net start com.docker.service")
	err := retry.Do(
		func() error {
			if err := h.Exec(c.DockerCommandf("ps")); err != nil {
				return fmt.Errorf("failed to run docker ps after restart: %w", err)
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(10),
	)
	if err != nil {
		return fmt.Errorf("failed to restart docker service: %w", err)
	}
	return nil
}

// ResolveInternalIP resolves internal ip from private interface.
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
		if iputil.IsValidAddress(addr) {
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
		return "", fmt.Errorf("failed to get IP address for interface %s: %w", iface, err)
	}
	return strings.TrimSpace(output), nil
}

// DockerCommandf accepts a printf-like template string and arguments
// and builds a command string for running the docker cli on the host.
func (c WindowsConfigurer) DockerCommandf(template string, args ...interface{}) string {
	return fmt.Sprintf("docker.exe %s", fmt.Sprintf(template, args...))
}

// ValidateLocalhost returns an error if "localhost" is not local on the host.
func (c WindowsConfigurer) ValidateLocalhost(h os.Host) error {
	err := h.Exec(ps.Cmd(`"$ips=[System.Net.Dns]::GetHostAddresses('localhost'); Get-NetIPAddress -IPAddress $ips"`))
	if err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host: %w", err)
	}
	return nil
}

// LocalAddresses returns a list of local addresses.
func (c WindowsConfigurer) LocalAddresses(h os.Host) ([]string, error) {
	output, err := h.ExecOutput(ps.Cmd(`(Get-NetIPAddress).IPV4Address`))
	if err != nil {
		return nil, fmt.Errorf("failed to get local addresses: %w", err)
	}
	var lines []string
	// bufio used to split lines on windows
	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines, nil
}

// CheckPrivilege returns an error if the user does not have admin access to the host.
func (c WindowsConfigurer) CheckPrivilege(h os.Host) error {
	privCheck := "\"$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent()); if (!$currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { $host.SetShouldExit(1) }\""

	if err := h.Exec(ps.Cmd(privCheck)); err != nil {
		return fmt.Errorf("user does not have administrator rights on the host: %w", err)
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host.
func (c WindowsConfigurer) AuthenticateDocker(h os.Host, user, pass, imageRepo string) error {
	// the --pasword-stdin seems to hang in windows
	if err := h.Exec(c.DockerCommandf("login -u %s -p %s %s", user, pass, imageRepo), exec.RedactString(user, pass), exec.AllowWinStderr()); err != nil {
		return fmt.Errorf("failed to login to docker registry: %w", err)
	}
	return nil
}

// UpdateEnvironment updates the hosts's environment variables.
func (c WindowsConfigurer) UpdateEnvironment(h os.Host, env map[string]string) error {
	if err := c.Windows.UpdateEnvironment(h, env); err != nil {
		return fmt.Errorf("failed updating the env: %w", err)
	}
	return nil
}

// CleanupEnvironment removes environment variable configuration.
func (c WindowsConfigurer) CleanupEnvironment(h os.Host, env map[string]string) error {
	if err := c.Windows.CleanupEnvironment(h, env); err != nil {
		return fmt.Errorf("failed cleaning the env: %w", err)
	}
	return nil
}

// ResolvePrivateInterface tries to find a private network interface.
func (c WindowsConfigurer) ResolvePrivateInterface(h os.Host) (string, error) {
	output, err := h.ExecOutput(ps.Cmd(`(Get-NetConnectionProfile -NetworkCategory Private | Select-Object -First 1).InterfaceAlias`))
	if err != nil || output == "" {
		output, err = h.ExecOutput(ps.Cmd(`(Get-NetConnectionProfile | Select-Object -First 1).InterfaceAlias`))
	}
	if err != nil || output == "" {
		return "", fmt.Errorf("failed to detect a private network interface, define the host privateInterface manually: %w", err)
	}
	return strings.TrimSpace(output), nil
}

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error.
func (c WindowsConfigurer) HTTPStatus(h os.Host, url string) (int, error) {
	log.Debugf("%s: requesting %s", h, url)
	output, err := h.ExecOutput(ps.Cmd(fmt.Sprintf(`[int][System.Net.WebRequest]::Create(%s).GetResponse().StatusCode`, ps.SingleQuote(url))))
	if err != nil {
		return -1, fmt.Errorf("failed to get HTTP status code: %w", err)
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("invalid response: %w", err)
	}
	return status, nil
}

// AuthorizeDocker does nothing on windows.
func (c WindowsConfigurer) AuthorizeDocker(_ os.Host) error {
	return nil
}

// CleanupLingeringMCR cleans up lingering MCR configuration files.
func (c WindowsConfigurer) CleanupLingeringMCR(h os.Host, dockerInfo common.DockerInfo) {
	dockerRootDir := constant.WindowsDefaultDockerRoot
	if dockerInfo.DockerRootDir != "" {
		dockerRootDir = dockerInfo.DockerRootDir
	}

	// Check if the Docker daemon configuration file exists
	exists, err := h.ExecOutput(ps.Cmd(fmt.Sprintf("Test-Path %s", ps.SingleQuote(c.MCRConfigPath()))))
	if err != nil {
		log.Errorf("error checking if Docker Daemon configuration file exists at %s: %v", c.MCRConfigPath(), err)
	}
	if exists == "True" {
		log.Infof("%s: MCR configuration file exists at %s", h, c.MCRConfigPath())
		var dockerDaemon common.DockerDaemonConfig
		dockerDaemonString, err := h.ExecOutput(ps.Cmd(fmt.Sprintf("Get-Content -Path %s", ps.SingleQuote(c.MCRConfigPath()))))
		if err != nil {
			dockerDaemon, err := c.GetDockerDaemonConfig(dockerDaemonString)
			if err != nil {
				log.Errorf("%s: error constructing dockerDaemon struct %+v: %s", h, dockerDaemon, err)
			}
		}
		if dockerDaemon.Root != "" {
			dockerRootDir = strings.TrimSpace(dockerDaemon.Root)
		}
	}

	c.attemptPathDelete(h, dockerRootDir)
}

func (c WindowsConfigurer) attemptPathDelete(h os.Host, path string) {
	// Remove a folder using PowerShell command.
	removeCommand := fmt.Sprintf("powershell Remove-Item -LiteralPath %s -Force -Recurse ", ps.SingleQuote(path))

	if err := h.Exec(removeCommand); err != nil {
		log.Debugf("%s: failed to remove %s: %s", h, path, err)
	} else {
		log.Infof("%s: removed %s successfully", h, path)
	}
}
