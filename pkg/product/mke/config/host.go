package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Mirantis/launchpad/pkg/configurer"
	common "github.com/Mirantis/launchpad/pkg/product/common/config"
	"github.com/Mirantis/launchpad/pkg/util/byteutil"
	retry "github.com/avast/retry-go"
	"github.com/creasty/defaults"
	"github.com/k0sproject/dig"
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/cmd"
	rigos "github.com/k0sproject/rig/v2/os"
	"github.com/k0sproject/rig/v2/protocol"
	"github.com/k0sproject/rig/v2/remotefs"
	"github.com/k0sproject/rig/v2/sudo"
	sloglogrus "github.com/samber/slog-logrus/v2"
	log "github.com/sirupsen/logrus"
)

// rigLogger bridges rig v2's slog-based logging into launchpad's logrus output.
// It wraps the logrus standard logger, so any level and hook changes applied
// there are reflected automatically. rig v2 has no global logger setter; the
// logger is injected per client via rig.WithLogger at Connect time.
var rigLogger = slog.New(sloglogrus.Option{
	Level:  slog.LevelDebug,
	Logger: log.StandardLogger(),
}.NewLogrusHandler())

// HostMetadata resolved metadata for host.
type HostMetadata struct {
	Hostname           string
	LongHostname       string
	InternalAddress    string
	MCRVersion         string
	MCRRestartRequired bool
	ImagesToUpload     []string
	TotalImageBytes    uint64
	MCRInstalled       bool // Indicates that in this run an MCR install has been executed (not that in installation has been discovered)
}

// MSRMetadata is metadata needed by MSR for configuration and is gathered at
// the GatherFacts phase and at the end of each configuration phase.
type MSRMetadata struct {
	Installed               bool
	InstalledVersion        string
	InstalledBootstrapImage string
	ReplicaID               string
}

type errs struct {
	errors []string
}

func (errors *errs) Count() int {
	return len(errors.errors)
}

func (errors *errs) Add(e string) {
	errors.errors = append(errors.errors, e)
}

func (errors *errs) Addf(template string, args ...interface{}) {
	errors.errors = append(errors.errors, fmt.Sprintf(template, args...))
}

func (errors *errs) String() string {
	if errors.Count() == 0 {
		return ""
	}

	return "- " + strings.Join(errors.errors, "\n- ")
}

// Host contains all the needed details to work with hosts.
type Host struct {
	rig.CompositeConfig `yaml:",inline"`
	*rig.Client         `yaml:"-"`

	Role             string            `yaml:"role" validate:"oneof=manager worker msr"`
	PrivateInterface string            `yaml:"privateInterface,omitempty" validate:"omitempty,gt=2"`
	DaemonConfig     dig.Mapping       `yaml:"mcrConfig,flow,omitempty" default:"{}"`
	Environment      map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks            common.Hooks      `yaml:"hooks,omitempty" validate:"dive,keys,oneof=apply reset,endkeys,dive,keys,oneof=before after,endkeys,omitempty"`
	ImageDir         string            `yaml:"imageDir,omitempty"`
	SudoDocker       bool              `yaml:"sudodocker"`
	SudoOverride     bool              `yaml:"sudooverride"`   // some customers can't allow the default rig connection sudo detection
	MCRUpgradeSkip   bool              `yaml:"mcrupgradeskip"` // don't upgrade this host when upgraing MCR (to allow upgrades in batches
	// SwarmAddressOverride, when set, is used as the advertise address for
	// Docker Swarm init and join operations instead of the discovered
	// InternalAddress. Use this in stretched/multi-DC environments where the
	// private NIC IP is not routable across DCs but the SSH/floating address is.
	SwarmAddressOverride string `yaml:"swarmAddress,omitempty"`

	Metadata    *HostMetadata  `yaml:"-"`
	MSRMetadata *MSRMetadata   `yaml:"-"`
	Configurer  HostConfigurer `yaml:"-"`
	OSRelease   *rigos.Release `yaml:"-"`
	Errors      errs           `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
// The alias type prevents recursion; the client is reset so Connect re-creates
// it with any updated connection config.
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type host Host
	yh := (*host)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	if err := defaults.Set(yh); err != nil {
		return fmt.Errorf("failed to set host defaults: %w", err)
	}

	if h.Client != nil {
		h.Disconnect()
		h.Client = nil
	}

	return nil
}

// Connect establishes the connection to the host, injecting launchpad's logger
// so that rig's internal logging is routed into logrus. When the host has
// sudooverride set, the sudo detection is bypassed and every privileged command
// is wrapped with sudo unconditionally.
func (h *Host) Connect(ctx context.Context) error {
	if h.Client == nil {
		opts := []rig.ClientOption{
			rig.WithConnectionFactory(&h.CompositeConfig),
			rig.WithLogger(rigLogger),
		}
		if ConfirmCommands {
			opts = append(opts, rig.WithConfirmFunc(confirmCommand))
		}
		if h.SudoOverride {
			opts = append(opts, rig.WithSudoProvider(func(runner cmd.Runner) (cmd.Runner, error) {
				return cmd.NewExecutor(runner, sudo.Sudo), nil
			}))
		}
		client, err := rig.NewClient(opts...)
		if err != nil {
			return fmt.Errorf("create rig client: %w", err)
		}
		h.Client = client
	}
	if err := h.Client.Connect(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	return nil
}

// String returns a human-readable description of the host, safe before Connect.
func (h *Host) String() string {
	if h.Client != nil {
		return h.Client.String()
	}
	return h.CompositeConfig.String()
}

// Protocol returns the host communication protocol family.
func (h *Host) Protocol() string {
	if h.Client != nil {
		return h.Client.Protocol()
	}
	if h.WinRM != nil {
		return "WinRM"
	}
	if bool(h.Localhost) {
		return "Local"
	}
	return "SSH"
}

// Address returns the host address, falling back to the configured connection
// address when the client is not yet connected.
func (h *Host) Address() string {
	if h.Client != nil {
		if addr := h.Client.Address(); addr != "" {
			return addr
		}
	}
	if h.SSH != nil && h.SSH.Address != "" {
		return h.SSH.Address
	}
	if h.WinRM != nil && h.WinRM.Address != "" {
		return h.WinRM.Address
	}
	if h.OpenSSH != nil && h.OpenSSH.Address != "" {
		return h.OpenSSH.Address
	}
	return "127.0.0.1"
}

// IsWindows returns true when the detected OS is Windows.
func (h *Host) IsWindows() bool {
	if h.Client != nil {
		return h.Client.IsWindows()
	}
	return h.WinRM != nil
}

// IsLocal returns true for localhost connections.
func (h *Host) IsLocal() bool {
	return h.Protocol() == "Local"
}

// IsSudoCommand is a particluar string command supposed to use Sudo.
func (h *Host) IsSudoCommand(cmd string) bool {
	if h.SudoDocker && (strings.HasPrefix(cmd, "docker") || strings.HasPrefix(cmd, "/usr/bin/docker")) {
		return true
	}
	return false
}

// runner returns the client used to execute cmd, using the sudo-decorated
// client clone when the command must run with elevated privileges.
func (h *Host) runner(cmd string) *rig.Client {
	if h.IsSudoCommand(cmd) {
		return h.Sudo()
	}
	return h.Client
}

// AuthorizeDocker if needed.
func (h *Host) AuthorizeDocker() error {
	if h.SudoDocker {
		log.Debugf("%s: not authorizing docker, as docker is meant to be run with sudo", h)
		return nil
	}

	return h.Configurer.AuthorizeDocker(h) //nolint:wrapcheck
}

// ExecStreams runs a command with the given stdin/stdout/stderr streams and
// returns a protocol.Waiter whose Wait blocks until the command exits. It is a
// thin wrapper over rig's cmd.Proc that adds the SudoDocker routing (see
// runner); callers that do not need that routing can use h.Proc directly.
func (h *Host) ExecStreams(cmd string, stdin io.Reader, stdout, stderr io.Writer, opts ...cmd.ExecOption) (protocol.Waiter, error) { //nolint:ireturn
	proc := h.runner(cmd).Proc(cmd)
	proc.Stdin = stdin
	proc.Stdout = stdout
	proc.Stderr = stderr
	waiter, err := proc.Start(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	return waiter, nil
}

// Exec runs a command on the host. It delegates to rig's runner, selecting the
// sudo-decorated client clone for commands that must run privileged (see runner
// / SudoDocker).
func (h *Host) Exec(cmd string, opts ...cmd.ExecOption) error {
	return h.runner(cmd).Exec(cmd, opts...) //nolint:wrapcheck
}

// ExecOutput runs a command on the host and returns its output, applying the
// same SudoDocker routing as Exec.
func (h *Host) ExecOutput(cmd string, opts ...cmd.ExecOption) (string, error) {
	return h.runner(cmd).ExecOutput(cmd, opts...) //nolint:wrapcheck
}

// ExecInteractive runs a command (or an interactive shell when cmd is empty)
// on the host, wired to the local standard streams.
func (h *Host) ExecInteractive(cmd string) error {
	if err := h.Client.ExecInteractive(context.Background(), cmd, os.Stdin, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("interactive exec failed: %w", err)
	}
	return nil
}

// ExecAll execs a slice of commands on the host.
func (h *Host) ExecAll(cmds []string) error {
	for _, cmd := range cmds {
		log.Infof("%s: Executing: %s", h, cmd)
		output, err := h.ExecOutput(cmd)
		if err != nil {
			log.Errorf("%s: %s", h, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h)))
			return fmt.Errorf("failed to execute step command: %w", err)
		}
		if strings.TrimSpace(output) != "" {
			log.Infof("%s: %s", h, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h)))
		}
	}
	return nil
}

var errAuthFailed = errors.New("authentication failed")

// AuthenticateDocker performs a docker login on the host using local REGISTRY_USERNAME
// and REGISTRY_PASSWORD when set.
func (h *Host) AuthenticateDocker(imageRepo string) error {
	if user := os.Getenv("REGISTRY_USERNAME"); user != "" {
		pass := os.Getenv("REGISTRY_PASSWORD")
		if pass == "" {
			return fmt.Errorf("%w: REGISTRY_PASSWORD not set", errAuthFailed)
		}

		log.Infof("%s: authenticating docker for image repo %s", h, imageRepo)
		if strings.HasPrefix(imageRepo, "docker.io/") { // docker.io is a special case for auth
			imageRepo = ""
		}
		if err := h.Configurer.AuthenticateDocker(h, user, pass, imageRepo); err != nil {
			return fmt.Errorf("%w: %s", errAuthFailed, err.Error())
		}
	}
	return nil
}

// SwarmAddress returns the address used for Swarm clustering.
// When swarmAddress is set in the host config it takes precedence over the
// discovered InternalAddress, allowing users in stretched/multi-DC environments
// to specify a routable floating address instead of the private NIC IP.
func (h *Host) SwarmAddress() string {
	addr := h.Metadata.InternalAddress
	if h.SwarmAddressOverride != "" {
		addr = h.SwarmAddressOverride
	}
	return fmt.Sprintf("%s:%d", addr, 2377)
}

// MCRVersion returns the current engine version installed on the host.
func (h *Host) MCRVersion() (string, error) {
	version, err := h.ExecOutput(h.Configurer.DockerCommandf(`version -f "{{.Server.Version}}"`))
	if err != nil {
		return "", fmt.Errorf("failed to get container runtime version: %w", err)
	}

	return version, nil
}

var errUnexpectedResponse = errors.New("unexpected response")

// CheckHTTPStatus will perform a web request to the url and return an error if the http status is not the expected.
// TLS certificate verification is skipped, since these checks target services
// with self-signed certificates (e.g. the MKE controllers).
func (h *Host) CheckHTTPStatus(url string, expected int) error {
	status, err := remotefs.HTTPStatusInsecure(context.Background(), h.FS(), url)
	if err != nil {
		return fmt.Errorf("failed to get http status: %w", err)
	}

	log.Debugf("%s: response code: %d, expected %d", h, status, expected)
	if status != expected {
		return fmt.Errorf("%w: code %d", errUnexpectedResponse, status)
	}

	return nil
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate.
func (h *Host) WriteFileLarge(src, dst string, fmo fs.FileMode) error {
	startTime := time.Now()
	stat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	size := stat.Size()
	if size < 0 {
		return fmt.Errorf("invalid file size: %d", size)
	}
	usize := uint64(size)

	log.Infof("%s: uploading %s to %s", h, byteutil.FormatBytes(usize), dst)

	if err := remotefs.Upload(h.FS(), src, dst, remotefs.WithPermissions(fmo)); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	duration := time.Since(startTime).Seconds()
	speed := float64(size) / duration
	log.Infof("%s: transferred %s in %.1f seconds (%s/s)", h, byteutil.FormatBytes(usize), duration, byteutil.FormatBytes(uint64(speed)))

	return nil
}

// Reconnect disconnects and reconnects the host's connection.
func (h *Host) Reconnect() error {
	h.Disconnect()

	log.Infof("%s: waiting for reconnection", h)
	err := retry.Do(
		func() error {
			if err := h.Connect(context.Background()); err != nil {
				return fmt.Errorf("failed to reconnect: %w", err)
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
	)
	if err != nil {
		return fmt.Errorf("retry count exceeded: %w", err)
	}
	return nil
}

// Reboot reboots the host and waits for it to become responsive.
func (h *Host) Reboot() error {
	log.Infof("%s: rebooting", h)
	if err := h.Configurer.Reboot(h); err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}
	log.Infof("%s: waiting for host to go offline", h)
	if err := h.waitForHost(false); err != nil {
		return err
	}
	h.Disconnect()

	log.Infof("%s: waiting for reconnection", h)
	if err := h.Reconnect(); err != nil {
		return fmt.Errorf("unable to reconnect after reboot: %w", err)
	}

	log.Infof("%s: waiting for host to become active", h)
	if err := h.waitForHost(true); err != nil {
		return err
	}

	if err := h.Reconnect(); err != nil {
		return fmt.Errorf("unable to reconnect after reboot: %w", err)
	}

	return nil
}

// ConfigureMCR writes the docker engine daemon.json and toggles the host Metadata MCRRestartRequired flag if changed.
func (h *Host) ConfigureMCR() error {
	if len(h.DaemonConfig) == 0 {
		return nil
	}

	cfgPath := h.Configurer.MCRConfigPath()

	oldJSON := make(dig.Mapping)

	if data, err := fs.ReadFile(h.Sudo().FS(), cfgPath); err == nil {
		log.Debugf("%s: parsing existing daemon.json", h)
		if err := json.Unmarshal(data, &oldJSON); err != nil {
			log.Debugf("%s: failed to parse existing MCR config: %s", h, err)
		}
	} else {
		log.Debugf("%s: failed to read existing MCR config: %s", h, err)
	}

	newJSONbytes, err := json.Marshal(h.DaemonConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal daemon.json config: %w", err)
	}

	newJSON := make(dig.Mapping)
	if err := json.Unmarshal(newJSONbytes, &newJSON); err != nil {
		return fmt.Errorf("failed to remarshal daemon.json: %w", err)
	}

	if reflect.DeepEqual(oldJSON, newJSON) {
		log.Debugf("%s: no changes for daemon.json", h)
		return nil
	}

	log.Debugf("%s: writing new daemon.json", h)

	if err := h.Sudo().FS().Remove(cfgPath); err != nil {
		log.Debugf("%s: failed to delete existing daemon.json: %s", h, err)
	}

	if err := h.Sudo().FS().WriteFile(cfgPath, newJSONbytes, fs.FileMode(0o600)); err != nil {
		return fmt.Errorf("failed to write daemon.json: %w", err)
	}

	if h.Metadata.MCRVersion != "" {
		h.Metadata.MCRRestartRequired = true
		log.Debugf("%s: host marked for mcr restart as MCR config was changed", h)
	}

	return nil
}

var errUnexpectedState = errors.New("unexpected state")

// when state is true wait for host to become active, when state is false, wait for connection to go down.
func (h *Host) waitForHost(state bool) error {
	err := retry.Do(
		func() error {
			err := h.Exec("echo")
			if !state && err == nil {
				return fmt.Errorf("%w: still online", errUnexpectedState)
			} else if state && err != nil {
				return fmt.Errorf("%w: still offline", errUnexpectedState)
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
	)
	if err != nil {
		return fmt.Errorf("failed to wait for host to go offline: %w", err)
	}
	return nil
}

var errUnsupportedOS = errors.New("unsupported OS")

// ResolveConfigurer assigns a rig-style configurer to the Host (see configurer/).
func (h *Host) ResolveConfigurer() error {
	if h.OSRelease == nil {
		release, err := h.OS()
		if err != nil {
			return fmt.Errorf("%w: OS detection failed: %w", errUnsupportedOS, err)
		}
		h.OSRelease = release
	}

	bf, ok := configurer.ResolveOSModule(h.OSRelease)
	if !ok {
		return fmt.Errorf("%w: %s", errUnsupportedOS, h.OSRelease.ID)
	}

	if c, ok := bf().(HostConfigurer); ok {
		h.Configurer = c

		return nil
	}

	return errUnsupportedOS
}
