package configurer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Mirantis/launchpad/pkg/constant"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/Mirantis/launchpad/pkg/util/iputil"
	"github.com/k0sproject/rig/v2/cmd"
	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/k0sproject/rig/v2/remotefs"
	"github.com/k0sproject/rig/v2/sh"
	"github.com/k0sproject/rig/v2/sh/shellescape"
	log "github.com/sirupsen/logrus"
)

const (
	// LinuxDockerLicenseFile filename for the docker license file on Linux machines.
	LinuxDockerLicenseFile = "docker.lic"
	// SbinPath is for adding sbin directories to current $PATH.
	SbinPath = `PATH=/usr/local/sbin:/usr/sbin:/sbin:$PATH`
)

var ErrLinuxMCRInstall = errors.New("failed to install MCR on linux")

// LinuxConfigurer is a generic linux host configurer.
type LinuxConfigurer struct {
	DockerConfigurer
}

// Pwd returns the current working directory of the session.
func (c LinuxConfigurer) Pwd(h Host) string {
	pwd, err := h.ExecOutput("pwd 2> /dev/null")
	if err != nil {
		return ""
	}
	return pwd
}

// IsContainer returns true if the host is actually a container.
func (c LinuxConfigurer) IsContainer(h Host) bool {
	return h.Exec("grep 'container=docker' /proc/1/environ 2> /dev/null") == nil
}

// FixContainer makes a container work like a real host.
func (c LinuxConfigurer) FixContainer(h Host) error {
	if err := h.Sudo().Exec("mount --make-rshared / 2> /dev/null"); err != nil {
		return fmt.Errorf("failed to mount / as rshared: %w", err)
	}
	return nil
}

// SELinuxEnabled is true when SELinux is enabled.
func (c LinuxConfigurer) SELinuxEnabled(h Host) bool {
	return h.Sudo().Exec("getenforce | grep -iq enforcing 2> /dev/null") == nil
}

// Reboot reboots the host.
func (c LinuxConfigurer) Reboot(h Host) error {
	if err := h.Sudo().Exec("shutdown --reboot 0 2> /dev/null"); err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}
	return nil
}

// InstallPackage installs the given packages using the host's package manager.
func (c LinuxConfigurer) InstallPackage(h Host, packages ...string) error {
	pm := h.Sudo().PackageManager()
	if err := pm.Update(context.Background()); err != nil {
		return fmt.Errorf("failed to update package sources: %w", err)
	}
	if err := pm.Install(context.Background(), packages...); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}
	return nil
}

// RemovePackage removes the given packages using the host's package manager.
func (c LinuxConfigurer) RemovePackage(h Host, packages ...string) error {
	if err := h.Sudo().PackageManager().Remove(context.Background(), packages...); err != nil {
		return fmt.Errorf("failed to remove packages: %w", err)
	}
	return nil
}

// StartService starts a service on the host.
func (c LinuxConfigurer) StartService(h Host, name string) error {
	svc, err := h.Sudo().Service(name)
	if err != nil {
		return fmt.Errorf("failed to resolve service %s: %w", name, err)
	}
	if err := svc.Start(context.Background()); err != nil {
		return fmt.Errorf("failed to start service %s: %w", name, err)
	}
	return nil
}

// StopService stops a service on the host.
func (c LinuxConfigurer) StopService(h Host, name string) error {
	svc, err := h.Sudo().Service(name)
	if err != nil {
		return fmt.Errorf("failed to resolve service %s: %w", name, err)
	}
	if err := svc.Stop(context.Background()); err != nil {
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}
	return nil
}

// RestartService restarts a service on the host.
func (c LinuxConfigurer) RestartService(h Host, name string) error {
	svc, err := h.Sudo().Service(name)
	if err != nil {
		return fmt.Errorf("failed to resolve service %s: %w", name, err)
	}
	if err := svc.Restart(context.Background()); err != nil {
		return fmt.Errorf("failed to restart service %s: %w", name, err)
	}
	return nil
}

// EnableService enables a service on the host.
func (c LinuxConfigurer) EnableService(h Host, name string) error {
	svc, err := h.Sudo().Service(name)
	if err != nil {
		return fmt.Errorf("failed to resolve service %s: %w", name, err)
	}
	if err := svc.Enable(context.Background()); err != nil {
		return fmt.Errorf("failed to enable service %s: %w", name, err)
	}
	return nil
}

// MCRConfigPath returns the configuration file path.
func (c LinuxConfigurer) MCRConfigPath() string {
	return "/etc/docker/daemon.json"
}

// InstallMCRLicense installs the MCR license file.
func (c LinuxConfigurer) InstallMCRLicense(h Host, lic string) error {
	// Use default docker root dir if not specified in docker info
	dockerRootDir := constant.LinuxDefaultDockerRoot

	// set the docker root dir from docker info if it exists
	if info, err := c.GetDockerInfo(h); err == nil && info != (commonconfig.DockerInfo{}) {
		dockerRootDir = info.DockerRootDir
	}

	licPath := filepath.Join(dockerRootDir, LinuxDockerLicenseFile)
	if err := h.Sudo().FS().WriteFile(licPath, []byte(lic), fs.FileMode(0o400)); err != nil {
		return fmt.Errorf("license write (linux); %w", err)
	}
	return nil
}

// EnableMCR enables and starts the Docker EE engine on Linux, assuming that you already have the repos setup.
func (c LinuxConfigurer) EnableMCR(h Host, _ commonconfig.MCRConfig) error {
	if err := c.EnableService(h, "docker"); err != nil {
		return fmt.Errorf("init manager could not enable docker-ee, %w", err)
	}
	if err := c.StartService(h, "docker"); err != nil {
		return fmt.Errorf("init manager could not start docker-ee, %w", err)
	}

	return nil
}

// RestartMCR restarts Docker EE engine.
func (c LinuxConfigurer) RestartMCR(h Host) error {
	if err := c.RestartService(h, "docker"); err != nil {
		return fmt.Errorf("restart docker service: %w", err)
	}
	return nil
}

// ResolveInternalIP resolves internal ip from private interface.
func (c LinuxConfigurer) ResolveInternalIP(h Host, privateInterface, publicIP string) (string, error) {
	output, err := h.ExecOutput(SbinPath + " " + sh.Command("ip", "-o", "addr", "show", "dev", privateInterface, "scope", "global"))
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
func (c LinuxConfigurer) DockerCommandf(template string, args ...any) string {
	return fmt.Sprintf("/usr/bin/docker %s", fmt.Sprintf(template, args...))
}

// ValidateLocalhost returns an error if "localhost" is not a local address.
func (c LinuxConfigurer) ValidateLocalhost(h Host) error {
	if err := h.Exec("ping -c 1 -w 1 localhost"); err != nil {
		return fmt.Errorf("hostname 'localhost' does not resolve to an address local to the host: %w", err)
	}
	return nil
}

// CheckPrivilege returns an error if the user does not have passwordless sudo enabled.
func (c LinuxConfigurer) CheckPrivilege(_ Host) error {
	return nil
}

// LocalAddresses returns a list of local addresses.
func (c LinuxConfigurer) LocalAddresses(h Host) ([]string, error) {
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
func (c LinuxConfigurer) AuthorizeDocker(h Host) error {
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

	if err := h.Sudo().Exec("usermod -aG docker $USER"); err != nil {
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
func (c LinuxConfigurer) AuthenticateDocker(h Host, user, pass, imageRepo string) error {
	if err := h.Exec(c.DockerCommandf("login -u %s --password-stdin %s", shellescape.Quote(user), imageRepo), cmd.StdinString(pass), cmd.Redact(user), cmd.Redact(pass)); err != nil {
		return fmt.Errorf("failed to login to the docker registry: %w", err)
	}
	return nil
}

// envKeyRegexp matches valid environment variable names: they must start with a
// letter or underscore and contain only letters, digits and underscores. This
// prevents keys with spaces or shell metacharacters from breaking (or being
// abused via) the /etc/environment and export steps.
var envKeyRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// UpdateEnvironment updates the hosts's environment variables.
func (c LinuxConfigurer) UpdateEnvironment(h Host, env map[string]string) error {
	fsys := h.Sudo().FS()
	for k, v := range env {
		if !envKeyRegexp.MatchString(k) {
			return fmt.Errorf("invalid environment variable key %q: must match %s", k, envKeyRegexp.String())
		}
		if strings.ContainsRune(v, '\n') {
			return fmt.Errorf("invalid environment variable value for key %q: must not contain newline", k)
		}
		patch := remotefs.ReplaceOrAppend(remotefs.ByPrefix(k+"="), fmt.Sprintf("%s=%s", k, v))
		if err := remotefs.PatchFile(fsys, "/etc/environment", []remotefs.Patch{patch}, remotefs.WithCreate(fs.FileMode(0o644))); err != nil {
			return fmt.Errorf("failed updating the env: %w", err)
		}
	}

	// Export the values into the current session environment using the
	// in-memory values with proper shell escaping.
	var export strings.Builder
	for k, v := range env {
		fmt.Fprintf(&export, "export %s=%s\n", k, shellescape.Quote(v))
	}
	if export.Len() > 0 {
		if err := h.Sudo().Exec(export.String()); err != nil {
			return fmt.Errorf("failed to update environment: %w", err)
		}
	}

	return c.ConfigureDockerProxy(h, env)
}

// CleanupEnvironment removes environment variable configuration.
func (c LinuxConfigurer) CleanupEnvironment(h Host, env map[string]string) error {
	if len(env) == 0 {
		return nil
	}
	fsys := h.Sudo().FS()
	patches := make([]remotefs.Patch, 0, len(env))
	for k := range env {
		patches = append(patches, remotefs.DeleteMatching(remotefs.ByPrefix(k+"=")))
	}
	if err := remotefs.PatchFile(fsys, "/etc/environment", patches, remotefs.WithCreate(fs.FileMode(0o644))); err != nil {
		return fmt.Errorf("failed cleaning the env: %w", err)
	}
	return nil
}

// ConfigureDockerProxy creates a docker systemd configuration for the proxy environment variables.
func (c LinuxConfigurer) ConfigureDockerProxy(h Host, env map[string]string) error {
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

	if err := h.Sudo().FS().MkdirAll(dir, fs.FileMode(0o755)); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}

	content := "[Service]\n"
	for k, v := range proxyenvs {
		content += fmt.Sprintf("Environment=\"%s=%s\"\n", shellescape.Quote(k), shellescape.Quote(v))
	}

	if err := h.Sudo().FS().WriteFile(cfg, []byte(content), fs.FileMode(0o600)); err != nil {
		return fmt.Errorf("failed to create %s: %w", cfg, err)
	}

	return nil
}

var errDetectPrivateInterface = errors.New("failed to detect a private network interface, define the host privateInterface manually")

// ResolvePrivateInterface tries to find a private network interface.
func (c LinuxConfigurer) ResolvePrivateInterface(h Host) (string, error) {
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

// CleanupLingeringMCR removes left over MCR files after Launchpad reset.
func (c LinuxConfigurer) CleanupLingeringMCR(h Host, dockerInfo commonconfig.DockerInfo) {
	// Use default docker root dir if not specified in docker info
	dockerRootDir := constant.LinuxDefaultDockerRoot
	dockerExecRootDir := constant.LinuxDefaultDockerExecRoot
	dockerDaemonPath := constant.LinuxDefaultDockerDaemonPath

	// set the docker root dir from docker info if it exists
	if dockerInfo != (commonconfig.DockerInfo{}) {
		dockerRootDir = dockerInfo.DockerRootDir
	}

	// https://docs.docker.com/config/daemon/
	if !h.Sudo().FS().FileExist(dockerDaemonPath) {
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

	dockerDaemonData, err := fs.ReadFile(h.Sudo().FS(), dockerDaemonPath)
	if err != nil {
		log.Debugf("%s: couldn't read the Docker Daemon config file %s: %s", h, dockerDaemonPath, err)
	}
	dockerConfig, err := c.GetDockerDaemonConfig(string(dockerDaemonData))
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
	if err := h.Sudo().Exec(fmt.Sprintf("umount %s", execRootNetnsUnmount)); err != nil {
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

func (c LinuxConfigurer) attemptPathSudoDelete(h Host, path string) {
	fileInfo, err := h.Sudo().FS().Stat(path)
	if err != nil {
		log.Debugf("%s: error getting file information for %s: %s", h, path, err)
		return
	}

	if !h.Sudo().FS().FileExist(path) {
		log.Infof("%s: file %s doesn't exist", h, path)
		return
	}

	command := fmt.Sprintf("rm %s", path)
	if fileInfo.IsDir() {
		command = fmt.Sprintf("rm -rf %s", path)
	}

	if err := h.Sudo().Exec(command); err != nil {
		log.Infof("%s: failed to remove %s: %s", h, path, err)
		return
	}
	log.Infof("%s: removed %s successfully", h, path)
}

var errAbort = errors.New("base os detected but version resolving failed")

// ResolveLinux resolves the OS release information for a linux host.
func ResolveLinux(h Host) (*rigos.Release, error) {
	release, ok := rigos.ResolveLinux(h)
	if !ok {
		return nil, fmt.Errorf("%w: unable to resolve linux OS release", errAbort)
	}
	return release, nil
}
