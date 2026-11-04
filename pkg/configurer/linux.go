package configurer

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
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
	// dockerBinaryLinux is the docker cli path used for remote commands.
	dockerBinaryLinux = "/usr/bin/docker"
)

// mcrPackage is the Mirantis Container Runtime package name. It is identical
// across the rpm and deb repositories on repos.mirantis.com.
const mcrPackage = "docker-ee"

// PackageManager identifies the package manager a Linux family installs MCR with.
type PackageManager int

const (
	// Yum is dnf/yum, on the enterprise linux family.
	Yum PackageManager = iota
	// AptGet is apt-get, on debian and ubuntu.
	AptGet
	// Zypper is zypper, on SLES.
	Zypper
)

// ErrUnknownPackageManager is returned for a PackageManager this package does
// not know how to build a command for.
var ErrUnknownPackageManager = errors.New("unknown package manager")

// MCRInstallCommand returns the command that installs the MCR runtime package.
//
// These commands are issued directly rather than through rig's InstallPackage,
// which hardcodes `<manager> install -y <packages>` and offers no way to pass
// additional arguments. SLES already had to bypass it for --allow-vendor-change
// (PRODENG-3623); installRecommends needs the same freedom on every family, so
// all three are built here instead, in one place, rather than three.
//
// When engineConfig.InstallRecommends is set, the manager is instructed to
// install recommended packages regardless of the host's configured default:
//
//   - yum/dnf: --setopt=install_weak_deps=True
//   - apt-get: -o APT::Install-Recommends=true, the configuration item behind
//     apt's --no-install-recommends flag
//   - zypper:  --recommends, "install also recommended packages in addition to
//     the required ones"
//
// Which packages those are is decided by the repository metadata, not here. See
// PRODENG-3641.
func MCRInstallCommand(manager PackageManager, engineConfig commonconfig.MCRConfig) (string, error) {
	switch manager {
	case Yum:
		args := []string{"yum", "install", "-y"}
		if engineConfig.InstallRecommends {
			args = append(args, "--setopt=install_weak_deps=True")
		}

		return strings.Join(append(args, mcrPackage), " "), nil
	case AptGet:
		// -q matches what rig's InstallPackage ran, so the default command is
		// unchanged. rig also ran `apt-get update` here; InstallMCR already does
		// that explicitly above, so it is not repeated.
		args := []string{"DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", "-q"}
		if engineConfig.InstallRecommends {
			args = append(args, "-o", "APT::Install-Recommends=true")
		}

		return strings.Join(append(args, mcrPackage), " "), nil
	case Zypper:
		// --allow-vendor-change is unconditional on SLES: cloud images ship a
		// SUSE-vendor containerd that the Mirantis packages must replace, and
		// zypper otherwise cancels non-interactively. See PRODENG-3623.
		args := []string{"zypper", "-n", "install", "-y", "--allow-vendor-change"}
		if engineConfig.InstallRecommends {
			args = append(args, "--recommends")
		}

		return strings.Join(append(args, mcrPackage), " "), nil
	default:
		return "", fmt.Errorf("%w: %d", ErrUnknownPackageManager, manager)
	}
}

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
	var reportedRoot string
	if info, err := c.GetDockerInfo(h); err == nil {
		reportedRoot = info.DockerRootDir
	}

	// Rootless detection has to run here too, not just in CleanupLingeringMCR:
	// MCR reads its subscription from its own data-root, so on a rootless host
	// the license has to follow the home data-root. Writing it to the default
	// root there would not merely relocate the file, it would leave the daemon
	// unlicensed. The resolution is shared so the two sites cannot drift.
	_, extraPrefixes := c.resolveDockerDaemonPath(h)

	licPath, trusted := licenseFilePath(reportedRoot, extraPrefixes)
	if !trusted && reportedRoot != "" {
		log.Warnf("%s: ignoring unsafe docker root path reported by the host: %q, writing the license to %s", h, reportedRoot, licPath)
	}

	if err := h.Sudo().FS().WriteFile(licPath, []byte(lic), fs.FileMode(0o400)); err != nil {
		return fmt.Errorf("license write (linux); %w", err)
	}
	return nil
}

// licenseFilePath resolves where the MCR license is written for a
// host-reported docker root, reporting whether that root was trusted.
//
// The root is reported by the host's docker daemon and the license is written
// into it as root, so it is validated first: an empty value would write to a
// relative path in the remote working directory, and a host-chosen one would
// drop a root-owned file anywhere on the filesystem. extraPrefixes carries the
// rootless home tree when rootless was detected, so a rootless data-root is
// honoured rather than rejected.
func licenseFilePath(reportedRoot string, extraPrefixes []string) (string, bool) {
	if isSafeRemotePath(reportedRoot, extraPrefixes...) {
		return filepath.Join(reportedRoot, LinuxDockerLicenseFile), true
	}

	return filepath.Join(constant.LinuxDefaultDockerRoot, LinuxDockerLicenseFile), false
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
//
// The result is executed by a login shell on the host, so every non-constant
// argument must already be quoted by the caller. Prefer DockerCmd, which
// quotes by construction.
func (c LinuxConfigurer) DockerCommandf(template string, args ...any) string {
	return fmt.Sprintf("%s %s", dockerBinaryLinux, fmt.Sprintf(template, args...))
}

// DockerCmd starts a docker command line for this host with arguments quoted
// for a POSIX shell.
func (c LinuxConfigurer) DockerCmd() *DockerCmd {
	return newDockerCmd(dockerBinaryLinux, shQuote)
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

	// rig's sudo wrapper runs the command through `sudo -- "$SHELL" -c '<cmd>'`, so any
	// variable in <cmd> is expanded inside the *elevated* shell after sudo's env_reset --
	// $USER/$LOGNAME/$HOME there resolve to root, not the SSH login user. $SUDO_USER is
	// explicitly exported by sudo across env_reset for exactly this case, so it still names
	// the real user.
	if err := h.Sudo().Exec("usermod -aG docker $SUDO_USER"); err != nil {
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
	dc := c.DockerCmd()
	loginCmd := dc.Raw("login -u").Arg(user).Raw("--password-stdin").Arg(imageRepo).String()
	// The username reaches the command line quoted; the password only goes in
	// over stdin. Patterns are built from both forms so redaction matches
	// whichever one is logged.
	if err := h.Exec(loginCmd,
		cmd.StdinString(pass),
		cmd.Redact(regexp.QuoteMeta(dc.Quote(user))),
		cmd.Redact(regexp.QuoteMeta(user)),
		cmd.Redact(regexp.QuoteMeta(pass)),
	); err != nil {
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

// HTTPStatus makes a HTTP GET request to the url and returns the status code or an error.
// TLS certificate verification is skipped, since these checks target services with
// self-signed certificates (e.g. the MKE controllers).
//
// This deliberately does not use rig's remotefs.HTTPStatusInsecure: that helper issues a
// HEAD request, which the MKE health endpoints do not answer with 200, so the health
// check would never pass. See PRODENG-3594.
func (c LinuxConfigurer) HTTPStatus(h Host, url string) (int, error) {
	log.Debugf("%s: requesting %s", h, url)
	output, err := h.ExecOutput(
		sh.Command("curl", "-kso", "/dev/null", "-w", "%{http_code}", "--", url),
		cmd.Sensitive(),
	)
	if err != nil {
		return -1, fmt.Errorf("failed to perform http request: %w", err)
	}
	status, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return -1, fmt.Errorf("invalid http response %q: %w", output, err)
	}

	return status, nil
}

// resolveDockerDaemonPath returns the docker daemon config file to read and
// any extra allowlist prefixes that host-reported paths may legitimately use.
//
// The extra prefixes are populated only when a rootless installation is
// actually detected: a rootless data-root lives under the user's home tree,
// which must not be writable or deletable on a rootful host. When the home
// tree cannot be resolved the allowlist stays rootful-only, because degrading
// to "rootless paths ignored" beats guessing a home directory.
func (c LinuxConfigurer) resolveDockerDaemonPath(h Host) (string, []string) {
	// https://docs.docker.com/config/daemon/
	if h.Sudo().FS().FileExist(constant.LinuxDefaultDockerDaemonPath) {
		return constant.LinuxDefaultDockerDaemonPath, nil
	}

	// Check if the default Rootless Docker daemon config file exists
	log.Debugf("%s: attempting to detect Rootless docker installation", h)
	// Extract the value from the xdgConfigHome environment variable
	xdgConfigHome, err := h.ExecOutput("echo $XDG_CONFIG_HOME")
	xdgConfigHome = strings.TrimSpace(xdgConfigHome)
	if err != nil || xdgConfigHome == "" {
		log.Debugf("%s: XDG_CONFIG_HOME not set, using default rootless daemon path %s", h, constant.LinuxDefaultRootlessDockerDaemonPath)
		return constant.LinuxDefaultRootlessDockerDaemonPath, nil
	}

	log.Debugf("%s: XDG_CONFIG_HOME set to %s", h, xdgConfigHome)
	daemonPath := path.Join(xdgConfigHome, "docker", "daemon.json")

	// A rootless data-root normally sits at $HOME/.local/share/docker, a
	// sibling of $XDG_CONFIG_HOME rather than a child, so the home tree is
	// what has to be admitted.
	home := rootlessHomeTree(xdgConfigHome)
	if home == "" {
		log.Debugf("%s: no rootless home tree resolved from %q, host-reported rootless paths will be ignored", h, xdgConfigHome)
		return daemonPath, nil
	}

	return daemonPath, []string{home}
}

// CleanupLingeringMCR removes left over MCR files after Launchpad reset.
func (c LinuxConfigurer) CleanupLingeringMCR(h Host, dockerInfo commonconfig.DockerInfo) {
	// Use default docker root dir if not specified in docker info
	dockerRootDir := constant.LinuxDefaultDockerRoot
	dockerExecRootDir := constant.LinuxDefaultDockerExecRoot

	dockerDaemonPath, extraPrefixes := c.resolveDockerDaemonPath(h)

	// dockerInfo.DockerRootDir is reported by the host's docker daemon and
	// flows into the sudo'd rm -rf below, so it gets the same validation as the
	// daemon.json values rather than being trusted. Keeping the default on an
	// empty value also matters without an attacker: "" would make the
	// path.Join calls below produce relative paths such as "cri-dockerd",
	// deleting from the remote working directory.
	if dockerInfo != (commonconfig.DockerInfo{}) {
		if isSafeRemotePath(dockerInfo.DockerRootDir, extraPrefixes...) {
			dockerRootDir = dockerInfo.DockerRootDir
		} else {
			log.Warnf("%s: ignoring unsafe docker root path reported by the host: %q", h, dockerInfo.DockerRootDir)
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
	// dockerConfig.Root/ExecRoot come from a host-controlled daemon.json;
	// anything that is not a clean absolute path in an expected location would
	// be executed via the sudo'd rm/umount below.
	if dockerConfig.Root != "" {
		if !isSafeRemotePath(dockerConfig.Root, extraPrefixes...) {
			log.Warnf("%s: ignoring unsafe docker root path from daemon config: %q", h, dockerConfig.Root)
		} else {
			dockerRootDir = dockerConfig.Root
		}
	}
	if dockerConfig.ExecRoot != "" {
		if !isSafeRemotePath(dockerConfig.ExecRoot, extraPrefixes...) {
			log.Warnf("%s: ignoring unsafe docker exec root path from daemon config: %q", h, dockerConfig.ExecRoot)
		} else {
			dockerExecRootDir = dockerConfig.ExecRoot
		}
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
	if err := h.Sudo().Exec(sh.Command("umount", execRootNetnsUnmount)); err != nil {
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

// safeRemotePathPrefixes are the locations a host-reported docker root or
// exec-root may legitimately live in on a rootful host. Syntax validation is
// not sufficient: "/etc" is a syntactically perfect absolute path, so a host
// reporting it as its data-root would otherwise earn a sudo rm -rf /etc.
var safeRemotePathPrefixes = []string{"/var/lib/", "/var/run/", "/run/"}

// safeRemotePathRegexp matches an absolute path with no shell metacharacters
// and no spaces. Values that reach sudo'd rm/umount commands are host-reported
// and must satisfy it.
var safeRemotePathRegexp = regexp.MustCompile(`^/[a-zA-Z0-9._/-]*$`)

// isSafeRemotePath reports whether a host-reported path is a plain absolute
// path in an expected location, and so may be used in a sudo'd remote command.
//
// extraPrefixes admits additional locations - in practice the rootless home
// tree - and must only be supplied when rootless was actually detected.
func isSafeRemotePath(remotePath string, extraPrefixes ...string) bool {
	if remotePath == "" || !safeRemotePathRegexp.MatchString(remotePath) {
		return false
	}

	// Non-canonical input is rejected rather than normalised: normalising would
	// make the guard's output depend on a transformation applied after
	// validation, so the value that was validated is not the value that gets
	// used. Rejecting covers "//", "/..", "/." and trailing slashes in one
	// check and makes the strings.LastIndex parent derivation in
	// CleanupLingeringMCR safe by construction: "/var/run/docker/" would
	// otherwise derive the parent "/var/run/docker" and delete the socket
	// inside the directory instead of beside it.
	if path.Clean(remotePath) != remotePath {
		return false
	}

	for _, prefix := range safeRemotePathPrefixes {
		if hasPathPrefix(remotePath, prefix) {
			return true
		}
	}
	for _, prefix := range extraPrefixes {
		if hasPathPrefix(remotePath, prefix) {
			return true
		}
	}

	return false
}

// hasPathPrefix reports whether p lies strictly inside dir, comparing whole
// path segments so that "/var/libfoo" does not satisfy a "/var/lib" prefix and
// dir itself is never treated as deletable.
func hasPathPrefix(p, dir string) bool {
	return strings.HasPrefix(p, strings.TrimSuffix(dir, "/")+"/")
}

// rootlessHomeTree derives the user home tree that a rootless data-root may
// live under from a host-reported XDG_CONFIG_HOME, returning "" when the value
// is not a plausible home config directory. XDG_CONFIG_HOME is host-controlled,
// so admitting its parent unchecked would let a host nominate any directory as
// deletable: "/etc/x" would yield "/etc".
func rootlessHomeTree(xdgConfigHome string) string {
	if xdgConfigHome == "" ||
		!safeRemotePathRegexp.MatchString(xdgConfigHome) ||
		path.Clean(xdgConfigHome) != xdgConfigHome {
		return ""
	}

	home := path.Dir(xdgConfigHome)
	if home == "/root" || path.Dir(home) == "/home" {
		return home
	}

	return ""
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

	command := sh.Command("rm", path)
	if fileInfo.IsDir() {
		command = sh.Command("rm", "-rf", path)
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
