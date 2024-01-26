package configurer

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
	escape "github.com/alessio/shellescape"
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
	pwd := c.riglinux.Pwd(h)
	base := path.Base(scriptPath)
	installer := pwd + "/" + base
	err := h.Upload(scriptPath, installer)
	if err != nil {
		log.Errorf("failed: %s", err.Error())
		return fmt.Errorf("upload %s to %s: %w", scriptPath, installer, err)
	}
	defer func() {
		if err := c.riglinux.DeleteFile(h, installer); err != nil {
			log.Warnf("failed to delete installer script: %s", err.Error())
		}
	}()

	cmd := fmt.Sprintf("DOCKER_URL=%s CHANNEL=%s VERSION=%s bash %s", engineConfig.RepoURL, engineConfig.Channel, engineConfig.Version, escape.Quote(installer))

	log.Infof("%s: running installer", h)

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
			if util.IsValidAddress(addr) {
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
	return fmt.Sprintf("docker %s", fmt.Sprintf(template, args...))
}

// ValidateLocalhost returns an error if "localhost" is not a local address.
func (c LinuxConfigurer) ValidateLocalhost(h os.Host) error {
	if err := h.Exec("sudo ping -c 1 -w 1 -r localhost"); err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host: %w", err)
	}
	return nil
}

// CheckPrivilege returns an error if the user does not have passwordless sudo enabled.
func (c LinuxConfigurer) CheckPrivilege(h os.Host) error {
	if err := h.Exec("sudo -n true"); err != nil {
		return fmt.Errorf("user does not have passwordless sudo access: %w", err)
	}

	return nil
}

// LocalAddresses returns a list of local addresses.
func (c LinuxConfigurer) LocalAddresses(h os.Host) ([]string, error) {
	output, err := h.ExecOutput("sudo hostname --all-ip-addresses")
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

	if err := h.Exec("[ -d $HOME/.docker ] && ([ ! -r $HOME/.docker ] || [ ! -w $HOME/.docker ]) && sudo chown -hR $USER:$(id -gn) $HOME/.docker"); err == nil {
		log.Warnf("%s: changed the owner of ~/.docker to be the current user", h)
	}

	if err := h.Exec("groups | grep -q docker"); err == nil {
		log.Debugf("%s: user already in the 'docker' group", h)
		return nil
	}

	if err := h.Exec("getent group docker"); err != nil {
		log.Debugf("%s: group 'docker' does not exist", h)
		return nil //nolint:nilerr
	}

	if err := h.Exec("sudo -i usermod -aG docker $USER"); err != nil {
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
	if err := h.Exec(c.DockerCommandf("login -u %s --password-stdin %s", user, imageRepo), exec.Stdin(pass), exec.RedactString(user, pass)); err != nil {
		return fmt.Errorf("failed to login to the docker registry: %w", err)
	}
	return nil
}

// LineIntoFile tries to find a matching line in a file and replace it with a new entry
// TODO refactor this into go because it's too magical.
func (c LinuxConfigurer) LineIntoFile(h os.Host, path, matcher, newLine string) error {
	if c.riglinux.FileExist(h, path) {
		err := h.Exec(fmt.Sprintf(`file=%s; match=%s; line=%s; sudo grep -q "${match}" "$file" && sudo sed -i "/${match}/c ${line}" "$file" || (echo "$line" | sudo tee -a "$file" > /dev/null)`, escape.Quote(path), escape.Quote(matcher), escape.Quote(newLine)))
		if err != nil {
			return fmt.Errorf("failed to update %s: %w", path, err)
		}
		return nil
	}
	if err := c.riglinux.WriteFile(h, path, newLine, "0600"); err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	return nil
}

// UpdateEnvironment updates the hosts's environment variables.
func (c LinuxConfigurer) UpdateEnvironment(h os.Host, env map[string]string) error {
	for k, v := range env {
		err := c.LineIntoFile(h, "/etc/environment", fmt.Sprintf("^%s=", k), fmt.Sprintf("%s=%s", k, v))
		if err != nil {
			return err
		}
	}

	// Update current environment from the /etc/environment
	err := h.Exec(`while read -r pair; do if [[ $pair == ?* && $pair != \#* ]]; then export "$pair" || exit 2; fi; done < /etc/environment`)
	if err != nil {
		return fmt.Errorf("failed to update current environment: %w", err)
	}

	return c.ConfigureDockerProxy(h, env)
}

// CleanupEnvironment removes environment variable configuration.
func (c LinuxConfigurer) CleanupEnvironment(h os.Host, env map[string]string) error {
	for k := range env {
		err := c.LineIntoFile(h, "/etc/environment", fmt.Sprintf("^%s=", k), "")
		if err != nil {
			return err
		}
	}
	// remove empty lines
	if err := h.Exec(`sudo sed -i '/^$/d' /etc/environment`); err != nil {
		return fmt.Errorf("failed to remove empty lines from /etc/environment: %w", err)
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

	err := h.Exec(fmt.Sprintf("sudo mkdir -p %s", dir))
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
		log.Debugf("%s attempting to detect Rootless docker installation", h)
		// Extract the value from the XDG_CONFIG_HOME environment variable
		XDG_CONFIG_HOME, err := h.ExecOutput("echo $XDG_CONFIG_HOME")
		if XDG_CONFIG_HOME != "" && err == nil {
			log.Debugf("%s XDG_CONFIG_HOME set to %s", h, XDG_CONFIG_HOME)
			dockerDaemonPath = path.Join(strings.TrimSpace(XDG_CONFIG_HOME), "docker", "daemon.json")
		} else {
			dockerDaemonPath = constant.LinuxDefaultRootlessDockerDaemonPath
			log.Debugf("%s XDG_CONFIG_HOME not set, using default rootless daemon path %s", h, dockerDaemonPath)
		}
	}

	dockerDaemonString, err := c.riglinux.ReadFile(h, dockerDaemonPath)
	if err != nil {
		log.Debugf("%s couldn't read the Docker Daemon config file %s: %s", h, dockerDaemonPath, err)
	}
	dockerConfig, err := c.GetDockerDaemonConfig(dockerDaemonString)
	if err != nil {
		log.Debugf("%s failed to create DockerDaemon config %s: %s", h, dockerConfig, err)
	}

	if dockerConfig.Root != "" {
		dockerRootDir = dockerConfig.Root
	}
	if dockerConfig.ExecRoot != "" {
		dockerExecRootDir = dockerConfig.ExecRoot
	}

	// /var/lib/ Root folder
	c.attemptPathDelete(h, dockerRootDir)
	if idx := strings.LastIndex(dockerRootDir, "/"); idx != -1 {
		dockerRootDir = dockerRootDir[:idx]
	}
	dockerCriPath := path.Join(dockerRootDir, "cri-dockerd")
	c.attemptPathDelete(h, dockerCriPath)

	// /var/run/ Exec-root folder
	execRootNetnsUnmount := path.Join(dockerExecRootDir, "netns/default")
	if err := h.Exec(fmt.Sprintf("sudo umount %s", execRootNetnsUnmount)); err != nil {
		log.Debugf("%s failed to umount %s: %s", h, execRootNetnsUnmount, err)
	}

	// Extras to delete if they exist
	if idx := strings.LastIndex(dockerExecRootDir, "/"); idx != -1 {
		dockerExecRootDir = dockerExecRootDir[:idx]
	}
	criDockerdMkeSock := path.Join(dockerExecRootDir, "cri-dockerd-mke.sock")
	c.attemptPathDelete(h, criDockerdMkeSock)

	dockerSock := path.Join(dockerExecRootDir, "docker.sock")
	c.attemptPathDelete(h, dockerSock)

	// /lib/systemd/system/ folder
	c.attemptPathDelete(h, "/lib/systemd/system/cri-dockerd-mke.service")
	c.attemptPathDelete(h, "/lib/systemd/system/cri-dockerd-mke.socket")
}

func (c LinuxConfigurer) attemptPathDelete(h os.Host, path string) {
	fileInfo, err := c.riglinux.Stat(h, path)
	if err != nil {
		log.Debugf("%s error getting file information for %s: %s", h, path, err)
	} else {
		command := fmt.Sprintf("sudo rm %s", path)
		if fileInfo.IsDir() {
			command = fmt.Sprintf("sudo rm -rf %s", path)
		}
		if c.riglinux.FileExist(h, path) {
			if err := h.Exec(command); err != nil {
				log.Infof("%s failed to remove %s: %s", h, path, err)
			}
			log.Infof("%s removed %s successfully", h, path)
		}
	}
}
