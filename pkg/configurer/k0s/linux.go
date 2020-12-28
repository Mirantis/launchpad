package configurer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	"github.com/Mirantis/mcc/pkg/util"

	log "github.com/sirupsen/logrus"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	escape "github.com/alessio/shellescape"
)

// LinuxConfigurer is a generic linux host configurer
type LinuxConfigurer struct {
	Host *api.Host
}

// SbinPath is for adding sbin directories to current $PATH
const SbinPath = `PATH=/usr/local/sbin:/usr/sbin:/sbin:$PATH`

// InstallK0s installs k0s binaries and sets up service either as systemd or openrc
func (c *LinuxConfigurer) InstallK0s(version string, k0sConfig *common.GenericHash) error {

	if c.Host.UploadBinary {
		if err := c.Host.Configurer.UploadK0s(version, k0sConfig); err != nil {
			return err
		}
	} else {
		if err := c.DownloadK0s(version, k0sConfig); err != nil {
			return err
		}
	}

	if err := c.K0sSetupService(); err != nil {
		return err
	}

	return nil
}

// DownloadK0s downloads k0s binaries
func (c *LinuxConfigurer) DownloadK0s(version string, k0sConfig *common.GenericHash) error {
	return c.Host.Exec(fmt.Sprintf("curl get.k0s.sh | K0S_VERSION=v%s sh", version))
}

// K0sSetupService sets up k0s as a systemd/openrc service
func (c *LinuxConfigurer) K0sSetupService() error {
	return c.Host.Exec(fmt.Sprintf("sudo k0s install --config %s --role %s", c.Host.Configurer.K0sConfigPath(), c.Host.Role))
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
	tempFile = escape.Quote(tempFile)

	err = c.Host.Exec(fmt.Sprintf("cat > %s && (sudo install -D -m %s %s %s || (rm %s; exit 1))", tempFile, permissions, tempFile, path, tempFile), exec.Stdin(data))
	if err != nil {
		return err
	}
	return nil
}

// ReadFile reads a files contents from the host.
func (c *LinuxConfigurer) ReadFile(path string) (string, error) {
	return c.Host.ExecWithOutput(fmt.Sprintf("sudo cat %s", escape.Quote(path)))
}

// DeleteFile deletes a file from the host.
func (c *LinuxConfigurer) DeleteFile(path string) error {
	return c.Host.Exec(fmt.Sprintf(`sudo rm -f %s`, escape.Quote(path)))
}

// FileExist checks if a file exists on the host
func (c *LinuxConfigurer) FileExist(path string) bool {
	return c.Host.Exec(fmt.Sprintf(`sudo test -e %s`, escape.Quote(path))) == nil
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

// ResolvePrivateInterface tries to find a private network interface
func (c *LinuxConfigurer) ResolvePrivateInterface() (string, error) {
	output, err := c.Host.ExecWithOutput(fmt.Sprintf(`%s; (ip route list scope global | grep -P "\b(172|10|192\.168)\.") || (ip route list | grep -m1 default)`, SbinPath))
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
	log.Debugf("%s: requesting %s", c.Host, url)
	output, err := c.Host.ExecWithOutput(fmt.Sprintf(`curl -kso /dev/null -w "%%{http_code}" "%s"`, url))
	if err != nil {
		return -1, err
	}
	status, err := strconv.Atoi(output)
	if err != nil {
		return -1, fmt.Errorf("%s: invalid response: %s", c.Host, err.Error())
	}

	return status, nil
}

// Pwd returns the current working directory of the session
func (c *LinuxConfigurer) Pwd() string {
	pwd, err := c.Host.ExecWithOutput("pwd")
	if err != nil {
		return ""
	}
	return pwd
}

// JoinPath joins a path
func (c *LinuxConfigurer) JoinPath(parts ...string) string {
	return strings.Join(parts, "/")
}

// RebootCommand returns a command string that will reboot the host
func (c *LinuxConfigurer) RebootCommand() string {
	return "sudo systemctl start reboot.target"
}
