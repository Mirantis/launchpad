package configurer

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"

	escape "al.essio.dev/pkg/shellescape"
	"github.com/Mirantis/launchpad/pkg/constant"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/Mirantis/launchpad/pkg/util/iputil"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"
	log "github.com/sirupsen/logrus"
)

// LinuxConfigurer is a generic linux host configurer.
type LinuxConfigurer struct {
	riglinux os.Linux
	DockerConfigurer
}

// SbinPath is for adding sbin directories to current $PATH.
const SbinPath = `PATH=/usr/local/sbin:/usr/sbin:/sbin:$PATH`

// MCRConfigPath returns the configuration file path.
func (c LinuxConfigurer) MCRConfigPath() string {
	return "/etc/docker/daemon.json"
}

// InstallMCR install MCR on Linux.
func (c LinuxConfigurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	base := path.Base(scriptPath)

	installScriptDir := engineConfig.InstallScriptRemoteDirLinux
	if installScriptDir == "" {
		installScriptDir = c.riglinux.Pwd(h)
	}

	_, err := h.ExecOutput(fmt.Sprintf("mkdir -p %s", installScriptDir))
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", installScriptDir, err)
	}

	installer := path.Join(installScriptDir, base)

	err = h.Upload(scriptPath, installer, fs.FileMode(0o640))
	if err != nil {
		log.Errorf("failed: %s", err.Error())
		return fmt.Errorf("upload %s to %s: %w", scriptPath, installer, err)
	}
	defer func() {
		if err := c.riglinux.DeleteFile(h, installer); err != nil {
			log.Warnf("failed to delete installer script: %s", err.Error())
		}
	}()

	envs := fmt.Sprintf("DOCKER_URL=%s CHANNEL=%s VERSION=%s ", engineConfig.RepoURL, engineConfig.Channel, engineConfig.Version)
	if engineConfig.AdditionalRuntimes != "" {
		envs += fmt.Sprintf("ADDITIONAL_RUNTIMES=%s ", engineConfig.AdditionalRuntimes)
	}
	if engineConfig.DefaultRuntime != "" {
		envs += fmt.Sprintf("DEFAULT_RUNTIME=%s ", engineConfig.DefaultRuntime)
	}
	cmd := envs + fmt.Sprintf("bash %s", escape.Quote(installer))

	log.Infof("%s: running installer", h)
	log.Debugf("%s: installer command: %s", h, cmd)

	if err := h.Exec(cmd); err != nil {
		return fmt.Errorf("run MCR installer: %w", err)
	}

	if err := c.riglinux.EnableService(h, "docker"); err != nil {
		return fmt.Errorf("enable docker service: %w", err)
	}

	if err := c.riglinux.StartService(h, "docker"); err != nil {
		return fmt.Errorf("start docker service: %w", err)
	}

	return nil
}

// RestartMCR restarts Docker EE engine.
func (c LinuxConfigurer) RestartMCR(h os.Host) error {
	if err := c.riglinux.RestartService(h, "docker"); err != nil {
		return fmt.Errorf("restart docker service: %w", err)
	}
	return nil
}

// StopMCR stop any running MCR components and dependencies (usually so that it can be uninstalled)
func (c LinuxConfigurer) StopMCR(h os.Host, engineConfig common.MCRConfig) error {
	info, getDockerError := c.GetDockerInfo(h)
	if engineConfig.Prune {
		defer c.CleanupLingeringMCR(h, info)
	}
	if getDockerError == nil {
		if err := h.Exec("docker system prune -f"); err != nil {
			return fmt.Errorf("prune docker: %w", err)
		}

		if err := c.riglinux.StopService(h, "docker"); err != nil {
			return fmt.Errorf("stop docker: %w", err)
		}

		if err := c.riglinux.StopService(h, "containerd"); err != nil {
			return fmt.Errorf("stop containerd: %w", err)
		}

		if err := h.Exec("yum remove -y docker-ee docker-ee-cli", exec.Sudo(h)); err != nil {
			return fmt.Errorf("remove docker-ee yum package: %w", err)
		}
	}

	return nil
}

// ResolveInternalIP resolves internal ip from private interface.
func (c LinuxConfigurer) ResolveInternalIP(h os.Host, privateInterface, publicIP string) (string, error) {
	output, err := h.ExecOutput(fmt.Sprintf("%s ip -o addr show dev %s scope global", SbinPath, privateInterface))
	if err != nil {
		return "", fmt.Errorf("%w: failed to find private interface with name %s: %s. Make sure you've set correct 'privateInterface' for the host in config", err, privateInterface, output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		items := strings.Fields(line)
		if len(items) < 4 {
			log.Debugf("not enough items in ip address line (%s), skipping...", items)
			continue
		}

		idx := strings.Index(items[3], "/")
		if idx == -1 {
			log.Debugf("no CIDR mask in ip address line (%s), skipping...", items)
			continue
		}
		addr := items[3][:idx]

		if addr != publicIP {
			log.Infof("%s: using %s as private IP", h, addr)
			if iputil.IsValidAddress(addr) {
				return addr, nil
			}
		}
	}

	log.Infof("%s: using %s as private IP", h, publicIP)

	return publicIP, nil
}

// DockerCommandf accepts a printf-like template string and arguments
// and builds a command string for running the docker cli on the host.
func (c LinuxConfigurer) DockerCommandf(template string, args ...interface{}) string {
	return fmt.Sprintf("/usr/bin/docker %s", fmt.Sprintf(template, args...))
}

// ValidateLocalhost returns an error if "localhost" is not a local address.
func (c LinuxConfigurer) ValidateLocalhost(h os.Host) error {
	if err := h.Exec("ping -c 1 -w 1 localhost"); err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host: %w", err)
	}
	return nil
}

// CheckPrivilege returns an error if the user does not have passwordless sudo enabled.
func (c LinuxConfigurer) CheckPrivilege(_ os.Host) error {
	return nil
}

// LocalAddresses returns a list of local addresses.
func (c LinuxConfigurer) LocalAddresses(h os.Host) ([]string, error) {
	output, err := h.ExecOutput("hostname --all-ip-addresses")
	if err != nil {
		return nil, fmt.Errorf("failed to get local addresses: %w", err)
	}

	return strings.Split(output, " "), nil
}

type reconnectable interface {
	String() string
	Reconnect() error
}

// AuthorizeDocker adds the current user to the docker group.
func (c LinuxConfigurer) AuthorizeDocker(h os.Host) error {
	if h.Exec(`[ "$(id -u)" = 0 ]`) == nil {
		log.Debugf("%s: current user is uid 0 - no need to authorize", h)
		return nil
	}

	if err := h.Exec("groups | grep -q docker"); err == nil {
		log.Debugf("%s: user already in the 'docker' group", h)
		return nil
	}

	if err := h.Exec("getent group docker"); err != nil {
		log.Debugf("%s: group 'docker' does not exist", h)
		return nil //nolint:nilerr
	}

	if err := h.Exec("usermod -aG docker $USER", exec.Sudo(h)); err != nil {
		return fmt.Errorf("failed to add the current user to the 'docker' group: %w", err)
	}

	log.Warnf("%s: added the current user to the 'docker' group", h)

	if h, ok := h.(reconnectable); ok {
		log.Infof("%s: reconnecting", h)
		if err := h.Reconnect(); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
	}

	if err := h.Exec("groups | grep -q docker"); err != nil {
		return fmt.Errorf("user is not in the 'docker' group: %w", err)
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host.
func (c LinuxConfigurer) AuthenticateDocker(h os.Host, user, pass, imageRepo string) error {
	if err := h.Exec(c.DockerCommandf("login -u %s --password-stdin %s", escape.Quote(user), imageRepo), exec.Stdin(pass), exec.RedactString(user, pass)); err != nil {
		return fmt.Errorf("failed to login to the docker registry: %w", err)
	}
	return nil
}

// UpdateEnvironment updates the hosts's environment variables.
func (c LinuxConfigurer) UpdateEnvironment(h os.Host, env map[string]string) error {
	if err := c.riglinux.UpdateEnvironment(h, env); err != nil {
		return fmt.Errorf("failed updating the env: %w", err)
	}
	return c.ConfigureDockerProxy(h, env)
}

// CleanupEnvironment removes environment variable configuration.
func (c LinuxConfigurer) CleanupEnvironment(h os.Host, env map[string]string) error {
	if err := c.riglinux.CleanupEnvironment(h, env); err != nil {
		return fmt.Errorf("failed cleaning the env: %w", err)
	}
	return nil
}

// ConfigureDockerProxy creates a docker systemd configuration for the proxy environment variables.
func (c LinuxConfigurer) ConfigureDockerProxy(h os.Host, env map[string]string) error {
	proxyenvs := make(map[string]string)

	for k, v := range env {
		if !strings.HasSuffix(k, "_PROXY") && !strings.HasSuffix(k, "_proxy") {
			continue
		}
		proxyenvs[k] = v
	}

	if len(proxyenvs) == 0 {
		return nil
	}

	dir := "/etc/systemd/system/docker.service.d"
	cfg := path.Join(dir, "http-proxy.conf")

	err := c.riglinux.MkDir(h, dir, exec.Sudo(h))
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	content := "[Service]\n"
	for k, v := range proxyenvs {
		content += fmt.Sprintf("Environment=\"%s=%s\"\n", escape.Quote(k), escape.Quote(v))
	}

	if err := c.riglinux.WriteFile(h, cfg, content, "0600"); err != nil {
		return fmt.Errorf("failed to create %s: %w", cfg, err)
	}

	return nil
}

var errDetectPrivateInterface = errors.New("failed to detect a private network interface, define the host privateInterface manually")

// ResolvePrivateInterface tries to find a private network interface.
func (c LinuxConfigurer) ResolvePrivateInterface(h os.Host) (string, error) {
	output, err := h.ExecOutput(fmt.Sprintf(`%s; (ip route list scope global | grep -P "\b(172|10|192\.168)\.") || (ip route list | grep -m1 default)`, SbinPath))
	if err != nil {
		return "", fmt.Errorf("%w: %w", errDetectPrivateInterface, err)
	}
	re := regexp.MustCompile(`\bdev (\w+)`)
	match := re.FindSubmatch([]byte(output))
	if len(match) == 0 {
		return "", fmt.Errorf("can't find 'dev' in output: %w", errDetectPrivateInterface)
	}
	return string(match[1]), nil
}

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error.
func (c LinuxConfigurer) HTTPStatus(h os.Host, url string) (int, error) {
	log.Debugf("%s: requesting %s", h, url)
	output, err := h.ExecOutput(fmt.Sprintf(`curl -kso /dev/null -w "%%{http_code}" "%s"`, url))
	if err != nil {
		return -1, fmt.Errorf("failed to perform http request: %w", err)
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("invalid http response: %w", err)
	}

	return status, nil
}

// CleanupLingeringMCR removes left over MCR files after Launchpad reset.
func (c LinuxConfigurer) CleanupLingeringMCR(h os.Host, dockerInfo common.DockerInfo) {
	// Use default docker root dir if not specified in docker info
	dockerRootDir := constant.LinuxDefaultDockerRoot
	dockerExecRootDir := constant.LinuxDefaultDockerExecRoot
	dockerDaemonPath := constant.LinuxDefaultDockerDaemonPath

	// set the docker root dir from docker info if it exists
	if dockerInfo != (common.DockerInfo{}) {
		dockerRootDir = dockerInfo.DockerRootDir
	}

	// https://docs.docker.com/config/daemon/
	if !c.riglinux.FileExist(h, dockerDaemonPath) {
		// Check if the default Rootless Docker daemon config file exists
		log.Debugf("%s: attempting to detect Rootless docker installation", h)
		// Extract the value from the xdgConfigHome environment variable
		xdgConfigHome, err := h.ExecOutput("echo $XDG_CONFIG_HOME")
		if xdgConfigHome != "" && err == nil {
			log.Debugf("%s: XDG_CONFIG_HOME set to %s", h, xdgConfigHome)
			dockerDaemonPath = path.Join(strings.TrimSpace(xdgConfigHome), "docker", "daemon.json")
		} else {
			dockerDaemonPath = constant.LinuxDefaultRootlessDockerDaemonPath
			log.Debugf("%s: XDG_CONFIG_HOME not set, using default rootless daemon path %s", h, dockerDaemonPath)
		}
	}

	dockerDaemonString, err := c.riglinux.ReadFile(h, dockerDaemonPath)
	if err != nil {
		log.Debugf("%s: couldn't read the Docker Daemon config file %s: %s", h, dockerDaemonPath, err)
	}
	dockerConfig, err := c.GetDockerDaemonConfig(dockerDaemonString)
	if err != nil {
		log.Debugf("%s: failed to create DockerDaemon config %s: %s", h, dockerConfig, err)
	}

	if dockerConfig.Root != "" {
		dockerRootDir = dockerConfig.Root
	}
	if dockerConfig.ExecRoot != "" {
		dockerExecRootDir = dockerConfig.ExecRoot
	}

	// /var/lib/ Root folder
	c.attemptPathSudoDelete(h, dockerRootDir)
	if idx := strings.LastIndex(dockerRootDir, "/"); idx != -1 {
		dockerRootDir = dockerRootDir[:idx]
	}
	c.attemptPathSudoDelete(h, path.Join(dockerRootDir, "cri-dockerd"))
	c.attemptPathSudoDelete(h, path.Join(dockerRootDir, "containerd"))
	c.attemptPathSudoDelete(h, path.Join(dockerRootDir, "kubelet"))

	// /var/run/ Exec-root folder
	execRootNetnsUnmount := path.Join(dockerExecRootDir, "netns/default")
	if err := h.Exec(fmt.Sprintf("umount %s", execRootNetnsUnmount), exec.Sudo(h)); err != nil {
		log.Debugf("%s: failed to umount %s: %s", h, execRootNetnsUnmount, err)
	}

	// Extras to delete if they exist
	if idx := strings.LastIndex(dockerExecRootDir, "/"); idx != -1 {
		dockerExecRootDir = dockerExecRootDir[:idx]
	}
	c.attemptPathSudoDelete(h, path.Join(dockerExecRootDir, "cri-dockerd-mke.sock"))
	c.attemptPathSudoDelete(h, path.Join(dockerExecRootDir, "docker.sock"))
	c.attemptPathSudoDelete(h, constant.LinuxDefaultDockerExecRoot)

	// /lib/systemd/system/ folder
	c.attemptPathSudoDelete(h, "/lib/systemd/system/cri-dockerd-mke.service")
	c.attemptPathSudoDelete(h, "/lib/systemd/system/cri-dockerd-mke.socket")
}

func (c LinuxConfigurer) attemptPathSudoDelete(h os.Host, path string) {
	fileInfo, err := c.riglinux.Stat(h, path, exec.Sudo(h))
	if err != nil {
		log.Debugf("%s: error getting file information for %s: %s", h, path, err)
		return
	}

	if !c.riglinux.FileExist(h, path) {
		log.Infof("%s: file %s doesn't exist", h, path)
		return
	}

	command := fmt.Sprintf("rm %s", path)
	if fileInfo.IsDir() {
		command = fmt.Sprintf("rm -rf %s", path)
	}

	if err := h.Exec(command, exec.Sudo(h)); err != nil {
		log.Infof("%s: failed to remove %s: %s", h, path, err)
		return
	}
	log.Infof("%s: removed %s successfully", h, path)
}
