package configurer

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/util"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	log "github.com/sirupsen/logrus"

	escape "github.com/alessio/shellescape"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	riglinux os.Linux
}

// SbinPath is for adding sbin directories to current $PATH
const SbinPath = `PATH=/usr/local/sbin:/usr/sbin:/sbin:$PATH`

// MCRConfigPath returns the configuration file path
func (c LinuxConfigurer) MCRConfigPath() string {
	return "/etc/docker/daemon.json"
}

// InstallMCR install MCR on Linux
func (c LinuxConfigurer) InstallMCR(h os.Host, scriptPath string, engineConfig common.MCRConfig) error {
	pwd := c.riglinux.Pwd(h)
	base := path.Base(scriptPath)
	installer := pwd + "/" + base
	err := h.Upload(scriptPath, installer)
	if err != nil {
		log.Errorf("failed: %s", err.Error())
		return err
	}
	defer c.riglinux.DeleteFile(h, installer)

	cmd := fmt.Sprintf("DOCKER_URL=%s CHANNEL=%s VERSION=%s bash %s", engineConfig.RepoURL, engineConfig.Channel, engineConfig.Version, escape.Quote(installer))

	log.Infof("%s: running installer", h)

	if err := h.Exec(cmd); err != nil {
		return err
	}

	if err := c.riglinux.EnableService(h, "docker"); err != nil {
		return err
	}

	return c.riglinux.StartService(h, "docker")
}

// RestartMCR restarts Docker EE engine
func (c LinuxConfigurer) RestartMCR(h os.Host) error {
	return c.riglinux.RestartService(h, "docker")
}

// ResolveInternalIP resolves internal ip from private interface
func (c LinuxConfigurer) ResolveInternalIP(h os.Host, privateInterface, publicIP string) (string, error) {
	output, err := h.ExecOutput(fmt.Sprintf("%s ip -o addr show dev %s scope global", SbinPath, privateInterface))
	if err != nil {
		return "", fmt.Errorf("failed to find private interface with name %s: %s. Make sure you've set correct 'privateInterface' for the host in config", privateInterface, output)
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		items := strings.Fields(line)
		if len(items) < 4 {
			log.Debugf("not enough items in ip address line (%s), skipping...", items)
			continue
		}
		addr := items[3][:strings.Index(items[3], "/")]
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
// and builds a command string for running the docker cli on the host
func (c LinuxConfigurer) DockerCommandf(template string, args ...interface{}) string {
	return fmt.Sprintf("docker %s", fmt.Sprintf(template, args...))
}

// ValidateLocalhost returns an error if "localhost" is not a local address
func (c LinuxConfigurer) ValidateLocalhost(h os.Host) error {
	if err := h.Exec("sudo ping -c 1 -w 1 -r localhost"); err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host")
	}
	return nil
}

// CheckPrivilege returns an error if the user does not have passwordless sudo enabled
func (c LinuxConfigurer) CheckPrivilege(h os.Host) error {
	if h.Exec("sudo -n true") != nil {
		return fmt.Errorf("user does not have passwordless sudo access")
	}

	return nil
}

// LocalAddresses returns a list of local addresses
func (c LinuxConfigurer) LocalAddresses(h os.Host) ([]string, error) {
	output, err := h.ExecOutput("sudo hostname --all-ip-addresses")
	if err != nil {
		return nil, err
	}

	return strings.Split(output, " "), nil
}

// AuthorizeDocker adds the current user to the docker group
func (c LinuxConfigurer) AuthorizeDocker(h os.Host) error {
	if h.Exec(`[ "$(id -u)" = 0 ]`) == nil {
		log.Debugf("%s: current user is uid 0 - no need to authorize", h)
		return nil
	}

	if err := h.Exec("[ -d $HOME/.docker ] && ([ ! -r $HOME/.docker ] || [ ! -w $HOME/.docker ]) && sudo chown -hR $USER:$USER $HOME/.docker"); err == nil {
		log.Warnf("%s: changed the owner of ~/.docker to be the current user", h)
	}

	if err := h.Exec("groups | grep -q docker"); err == nil {
		log.Debugf("%s: user already in the 'docker' group", h)
		return nil
	}

	if err := h.Exec("getent group docker"); err != nil {
		log.Debugf("%s: group 'docker' does not exist", h)
		return nil
	}

	if err := h.Exec("sudo usermod -aG docker $USER"); err != nil {
		return err
	}

	log.Warnf("%s: added the current user to the 'docker' group", h)

	log.Debugf("%s: reloading groups for the current session by trying to switch to the 'docker' group", h)
	og, err := h.ExecOutput("id -gn")
	if err != nil {
		return err
	}

	if err := h.Exec("newgrp docker"); err != nil {
		return err
	}

	if err := h.Execf("newgrp %s", og); err != nil {
		return err
	}

	if err := h.Exec("groups | grep -q docker"); err != nil {
		return fmt.Errorf("user is not in the 'docker' group")
	}

	return nil
}

// AuthenticateDocker performs a docker login on the host
func (c LinuxConfigurer) AuthenticateDocker(h os.Host, user, pass, imageRepo string) error {
	return h.Exec(c.DockerCommandf("login -u %s --password-stdin %s", user, imageRepo), exec.Stdin(pass), exec.RedactString(user, pass))
}

// LineIntoFile tries to find a matching line in a file and replace it with a new entry
// TODO refactor this into go because it's too magical.
func (c LinuxConfigurer) LineIntoFile(h os.Host, path, matcher, newLine string) error {
	if c.riglinux.FileExist(h, path) {
		err := h.Exec(fmt.Sprintf(`file=%s; match=%s; line=%s; sudo grep -q "${match}" "$file" && sudo sed -i "/${match}/c ${line}" "$file" || (echo "$line" | sudo tee -a "$file" > /dev/null)`, escape.Quote(path), escape.Quote(matcher), escape.Quote(newLine)))
		if err != nil {
			return err
		}
		return nil
	}
	return c.riglinux.WriteFile(h, path, newLine, "0700")
}

// UpdateEnvironment updates the hosts's environment variables
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
		return err
	}

	return c.ConfigureDockerProxy(h, env)
}

// CleanupEnvironment removes environment variable configuration
func (c LinuxConfigurer) CleanupEnvironment(h os.Host, env map[string]string) error {
	for k := range env {
		err := c.LineIntoFile(h, "/etc/environment", fmt.Sprintf("^%s=", k), "")
		if err != nil {
			return err
		}
	}
	// remove empty lines
	return h.Exec(`sudo sed -i '/^$/d' /etc/environment`)
}

// ConfigureDockerProxy creates a docker systemd configuration for the proxy environment variables
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
		return err
	}

	content := "[Service]\n"
	for k, v := range proxyenvs {
		content += fmt.Sprintf("Environment=\"%s=%s\"\n", escape.Quote(k), escape.Quote(v))
	}

	return c.riglinux.WriteFile(h, cfg, content, "0600")
}

// ResolvePrivateInterface tries to find a private network interface
func (c LinuxConfigurer) ResolvePrivateInterface(h os.Host) (string, error) {
	output, err := h.ExecOutput(fmt.Sprintf(`%s; (ip route list scope global | grep -P "\b(172|10|192\.168)\.") || (ip route list | grep -m1 default)`, SbinPath))
	if err == nil {
		re := regexp.MustCompile(`\bdev (\w+)`)
		match := re.FindSubmatch([]byte(output))
		if len(match) > 0 {
			return string(match[1]), nil
		}
		err = fmt.Errorf("can't find 'dev' in output")
	}

	return "", fmt.Errorf("failed to detect a private network interface, define the host privateInterface manually (%s)", err.Error())
}

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error
func (c LinuxConfigurer) HTTPStatus(h os.Host, url string) (int, error) {
	log.Debugf("%s: requesting %s", h, url)
	output, err := h.ExecOutput(fmt.Sprintf(`curl -kso /dev/null -w "%%{http_code}" "%s"`, url))
	if err != nil {
		return -1, err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("%s: invalid response: %s", h, err.Error())
	}

	return status, nil
}
