package configurer

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/util"

	"github.com/Mirantis/mcc/pkg/api"
	log "github.com/sirupsen/logrus"

	escape "github.com/alessio/shellescape"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	Host *api.Host
}

// InstallEngine install Docker EE engine on Linux
func (c *LinuxConfigurer) InstallEngine(engineConfig *api.EngineConfig) error {
	if len(c.Host.DaemonConfig) > 0 {
		daemonJSONData, err := json.Marshal(c.Host.DaemonConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal daemon json config: %w", err)
		}

		cfg := "/etc/docker/daemon.json"
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
	cmd := fmt.Sprintf("DOCKER_URL=%s CHANNEL=%s VERSION=%s bash -s", engineConfig.RepoURL, engineConfig.Channel, engineConfig.Version)
	err := c.Host.Exec(cmd, exec.Stdin(*c.Host.Metadata.EngineInstallScript))
	if err != nil {
		return err
	}

	err = c.Host.Exec("sudo systemctl enable docker")
	if err != nil {
		return err
	}

	err = c.Host.Exec("sudo systemctl start docker")
	if err != nil {
		return err
	}
	return nil
}

// RestartEngine restarts Docker EE engine
func (c *LinuxConfigurer) RestartEngine() error {
	return c.Host.Exec("sudo systemctl restart docker")
}

// ResolveHostname resolves the short hostname
func (c *LinuxConfigurer) ResolveHostname() string {
	hostname, _ := c.Host.ExecWithOutput("hostname -s")

	return hostname
}

// ResolveLongHostname resolves the FQDN (long) hostname
func (c *LinuxConfigurer) ResolveLongHostname() string {
	longHostname, _ := c.Host.ExecWithOutput("hostname")

	return longHostname
}

// ResolveInternalIP resolves internal ip from private interface
func (c *LinuxConfigurer) ResolveInternalIP() (string, error) {
	output, err := c.Host.ExecWithOutput(fmt.Sprintf("ip -o addr show dev %s scope global", c.Host.PrivateInterface))
	if err != nil {
		return "", fmt.Errorf("failed to find private interface with name %s: %s. Make sure you've set correct 'privateInterface' for the host in config", c.Host.PrivateInterface, output)
	}
	return c.ParseInternalIPFromIPOutput(output)
}

// ParseInternalIPFromIPOutput parses internal ip from ip command output
func (c *LinuxConfigurer) ParseInternalIPFromIPOutput(output string) (string, error) {
	lines := strings.Split(output, "\r\n")
	for _, line := range lines {
		items := strings.Fields(line)
		if len(items) < 4 {
			log.Debugf("not enough items in ip address line (%s), skipping...", items)
			continue
		}
		addrItems := strings.Split(items[3], "/")
		if addrItems[0] != c.Host.Address {
			if util.IsValidAddress(addrItems[0]) {
				return addrItems[0], nil
			}

			return "", fmt.Errorf("found address %s for interface %s but it does not seem to be valid address", addrItems[0], c.Host.PrivateInterface)
		}
	}
	// FIXME If we get this far should we just bail out with error!?!?
	return c.Host.Address, nil
}

// IsContainerized checks if host is actually a container
func (c *LinuxConfigurer) IsContainerized() bool {
	err := c.Host.Exec("grep 'container=docker' /proc/1/environ")
	if err != nil {
		return false
	}
	return true
}

// FixContainerizedHost configures host if host is containerized environment
func (c *LinuxConfigurer) FixContainerizedHost() error {
	if c.IsContainerized() {
		return c.Host.Exec("sudo mount --make-rshared /")
	}
	return nil
}

// DockerCommandf accepts a printf-like template string and arguments
// and builds a command string for running the docker cli on the host
func (c *LinuxConfigurer) DockerCommandf(template string, args ...interface{}) string {
	return fmt.Sprintf("sudo docker %s", fmt.Sprintf(template, args...))
}

// ValidateFacts validates all the collected facts so we're sure we can proceed with the installation
func (c *LinuxConfigurer) ValidateFacts() error {
	err := c.Host.Exec("ping -c 1 -w 1 -r localhost")
	if err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host")
	}

	localAddresses, err := c.getHostLocalAddresses()
	if err != nil {
		return fmt.Errorf("failed to find host local addresses: %w", err)
	}

	if !util.StringSliceContains(localAddresses, c.Host.Metadata.InternalAddress) {
		return fmt.Errorf("discovered private address %s does not seem to be a node local address (%s). Make sure you've set correct 'privateInterface' for the host in config", c.Host.Metadata.InternalAddress, strings.Join(localAddresses, ","))
	}

	return nil
}

// CheckPrivilege returns an error if the user does not have passwordless sudo enabled
func (c *LinuxConfigurer) CheckPrivilege() error {
	if c.Host.Exec("sudo -n true") != nil {
		return fmt.Errorf("user does not have passwordless sudo access")
	}

	return nil
}

// SELinuxEnabled is SELinux enabled
func (c *LinuxConfigurer) SELinuxEnabled() bool {
	output, err := c.Host.ExecWithOutput("sudo getenforce")
	if err != nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(output)) == "enforcing"
}

func (c *LinuxConfigurer) getHostLocalAddresses() ([]string, error) {
	output, err := c.Host.ExecWithOutput("sudo hostname --all-ip-addresses")
	if err != nil {
		return nil, err
	}

	return strings.Split(output, " "), nil
}

// AuthenticateDocker performs a docker login on the host
func (c *LinuxConfigurer) AuthenticateDocker(user, pass, imageRepo string) error {
	return c.Host.Exec(c.DockerCommandf("login -u %s --password-stdin %s", user, imageRepo), exec.Stdin(pass), exec.Redact(fmt.Sprintf("(%s|%s)", regexp.QuoteMeta(user), regexp.QuoteMeta(pass))))
}

// WriteFile writes file to host with given contents. Do not use for large files.
func (c *LinuxConfigurer) WriteFile(path string, data string, permissions string) error {
	if data == "" {
		return fmt.Errorf("empty content in WriteFile to %s", path)
	}

	if path == "" {
		return fmt.Errorf("empty path in WriteFile")
	}

	tempFile, err := c.Host.ExecWithOutput("mktemp")
	if err != nil {
		return err
	}

	err = c.Host.Exec(fmt.Sprintf("cat > %s && (sudo install -D -m %s %s %s || (rm %s; exit 1))", tempFile, permissions, tempFile, path, tempFile), exec.Stdin(data))
	if err != nil {
		return err
	}
	return nil
}

// ReadFile reads a files contents from the host.
func (c *LinuxConfigurer) ReadFile(path string) (string, error) {
	return c.Host.ExecWithOutput(fmt.Sprintf("sudo cat \"%s\"", path))
}

// DeleteFile deletes a file from the host.
func (c *LinuxConfigurer) DeleteFile(path string) error {
	return c.Host.Exec(fmt.Sprintf(`sudo rm -f "%s"`, path))
}

// FileExist checks if a file exists on the host
func (c *LinuxConfigurer) FileExist(path string) bool {
	return c.Host.Exec(fmt.Sprintf(`sudo test -e "%s"`, path)) == nil
}

// LineIntoFile tries to find a matching line in a file and replace it with a new entry
// TODO refactor this into go because it's too magical.
func (c *LinuxConfigurer) LineIntoFile(path, matcher, newLine string) error {
	if c.FileExist(path) {
		err := c.Host.Exec(fmt.Sprintf(`file=%s; match=%s; line=%s; sudo grep -q "${match}" "$file" && sudo sed -i "/${match}/c ${line}" "$file" || (echo "$line" | sudo tee -a "$file" > /dev/null)`, escape.Quote(path), escape.Quote(matcher), escape.Quote(newLine)))
		if err != nil {
			return err
		}
		return nil
	}
	return c.WriteFile(path, newLine, "0700")
}

// UpdateEnvironment updates the hosts's environment variables
func (c *LinuxConfigurer) UpdateEnvironment() error {
	for k, v := range c.Host.Environment {
		err := c.LineIntoFile("/etc/environment", fmt.Sprintf("^%s=", k), fmt.Sprintf("%s=%s", k, v))
		if err != nil {
			return err
		}
	}

	// Update current environment from the /etc/environment
	err := c.Host.Exec(`while read -r pair; do if [[ $pair == ?* && $pair != \#* ]]; then export "$pair" || exit 2; fi; done < /etc/environment`)
	if err != nil {
		return err
	}

	return c.ConfigureDockerProxy()
}

// CleanupEnvironment removes environment variable configuration
func (c *LinuxConfigurer) CleanupEnvironment() error {
	for k := range c.Host.Environment {
		err := c.LineIntoFile("/etc/environment", fmt.Sprintf("^%s=", k), "")
		if err != nil {
			return err
		}
	}
	// remove empty lines
	return c.Host.Exec(`sudo sed -i '/^$/d' /etc/environment`)
}

// ConfigureDockerProxy creates a docker systemd configuration for the proxy environment variables
func (c *LinuxConfigurer) ConfigureDockerProxy() error {
	proxyenvs := make(map[string]string)

	for k, v := range c.Host.Environment {
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

	err := c.Host.Exec(fmt.Sprintf("sudo mkdir -p %s", dir))
	if err != nil {
		return err
	}

	content := "[Service]\n"
	for k, v := range proxyenvs {
		content += fmt.Sprintf("Environment=\"%s=%s\"\n", escape.Quote(k), escape.Quote(v))
	}

	return c.WriteFile(cfg, content, "0600")
}

// ResolvePrivateInterface tries to find a private network interface
func (c *LinuxConfigurer) ResolvePrivateInterface() (string, error) {
	output, err := c.Host.ExecWithOutput(`(ip route list scope global | grep -P "\b(172|10|192\.168)\.") || (ip route list | grep -m1 default)`)
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
func (c *LinuxConfigurer) HTTPStatus(url string) (int, error) {
	log.Debugf("%s: requesting %s", c.Host.Address, url)
	output, err := c.Host.ExecWithOutput(fmt.Sprintf(`curl -kso /dev/null -w "%%{http_code}" "%s"`, url))
	if err != nil {
		return -1, err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("%s: invalid response: %s", c.Host.Address, err.Error())
	}

	return status, nil
}
