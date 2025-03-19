package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"time"

	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	"github.com/Mirantis/launchpad/pkg/util/byteutil"
	retry "github.com/avast/retry-go"
	"github.com/creasty/defaults"
	"github.com/k0sproject/dig"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/exec"
	"github.com/k0sproject/rig/os/registry"
	log "github.com/sirupsen/logrus"
)

// HostMetadata resolved metadata for host.
type HostMetadata struct {
	Hostname           string
	LongHostname       string
	InternalAddress    string
	MCRVersion         string
	MCRInstallScript   string
	MCRRestartRequired bool
	ImagesToUpload     []string
	TotalImageBytes    uint64
	MCRInstalled       bool
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
	rig.Connection `yaml:",inline"`

	Role             string            `yaml:"role" validate:"oneof=manager worker msr"`
	PrivateInterface string            `yaml:"privateInterface,omitempty" validate:"omitempty,gt=2"`
	DaemonConfig     dig.Mapping       `yaml:"mcrConfig,flow,omitempty" default:"{}"`
	Environment      map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks            common.Hooks      `yaml:"hooks,omitempty" validate:"dive,keys,oneof=apply reset,endkeys,dive,keys,oneof=before after,endkeys,omitempty"`
	ImageDir         string            `yaml:"imageDir,omitempty"`
	SudoDocker       bool              `yaml:"sudodocker"`
	SudoOverride     bool              `yaml:"sudooverride"` // some customers can't allow the default rig connection sudo detection

	Metadata    *HostMetadata  `yaml:"-"`
	MSRMetadata *MSRMetadata   `yaml:"-"`
	Configurer  HostConfigurer `yaml:"-"`
	Errors      errs           `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type host Host
	yh := (*host)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	if yh.SSH != nil && yh.SSH.HostKey != "" {
		log.Warnf("%s: spec.hosts[*].ssh.hostKey is deprecated, please use ssh known hosts file instead (.ssh/config, SSH_KNOWN_HOSTS)", h)
	}

	if err := defaults.Set(yh); err != nil {
		return fmt.Errorf("failed to set host defaults: %w", err)
	}
	return nil
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

// AuthorizeDocker if needed.
func (h *Host) AuthorizeDocker() error {
	if h.SudoDocker {
		log.Debugf("%s: not authorizing docker, as docker is meant to be run with sudo", h)
		return nil
	}

	return h.Configurer.AuthorizeDocker(h) //nolint:wrapcheck
}

func (h *Host) sudoCommandOptions(cmd string, opts []exec.Option) []exec.Option {
	if h.IsSudoCommand(cmd) {
		log.Debugf("%s: Exec is getting SUDOed as the command is in the host sudo list: %s", h, cmd)
		opts = append(opts, exec.Sudo(h))
	}
	return opts
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

// ExecStreams executes a command on the remote host and uses the passed in streams for stdin, stdout and stderr. It returns a Waiter with a .Wait() function that
// blocks until the command finishes and returns an error if the exit code is not zero.
func (h *Host) ExecStreams(cmd string, stdin io.ReadCloser, stdout, stderr io.Writer, opts ...exec.Option) (exec.Waiter, error) { //nolint:ireturn
	return h.Connection.ExecStreams(cmd, stdin, stdout, stderr, h.sudoCommandOptions(cmd, opts)...) //nolint:wrapcheck
}

// Exec runs a command on the host.
func (h *Host) Exec(cmd string, opts ...exec.Option) error {
	return h.Connection.Exec(cmd, h.sudoCommandOptions(cmd, opts)...) //nolint:wrapcheck
}

// ExecOutput runs a command on the host and returns the output as a String.
func (h *Host) ExecOutput(cmd string, opts ...exec.Option) (string, error) {
	return h.Connection.ExecOutput(cmd, h.sudoCommandOptions(cmd, opts)...) //nolint:wrapcheck
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

// SwarmAddress determines the swarm address for the host.
func (h *Host) SwarmAddress() string {
	return fmt.Sprintf("%s:%d", h.Metadata.InternalAddress, 2377)
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
func (h *Host) CheckHTTPStatus(url string, expected int) error {
	status, err := h.Configurer.HTTPStatus(h, url)
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
	usize := uint64(size) //nolint: gosec

	log.Infof("%s: uploading %s to %s", h, byteutil.FormatBytes(usize), dst)

	if err := h.Connection.Upload(src, dst, fmo); err != nil {
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
			if err := h.Connect(); err != nil {
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

	if f, err := h.Configurer.ReadFile(h, cfgPath); err == nil {
		log.Debugf("%s: parsing existing daemon.json", h)
		if err := json.Unmarshal([]byte(f), &oldJSON); err != nil {
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

	daemonJSONContent := string(newJSONbytes)

	if err := h.Configurer.DeleteFile(h, cfgPath); err != nil {
		log.Debugf("%s: failed to delete existing daemon.json: %s", h, err)
	}

	if err := h.Configurer.WriteFile(h, cfgPath, daemonJSONContent, "0600"); err != nil {
		return fmt.Errorf("failed to write daemon.json: %w", err)
	}

	if h.Metadata.MCRVersion != "" {
		h.Metadata.MCRRestartRequired = true
		log.Debugf("%s: host marked for mcr restart", h)
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
	bf, err := registry.GetOSModuleBuilder(*h.OSVersion)
	if err != nil {
		return fmt.Errorf("%w: failed to get OS module builder: %w", errUnsupportedOS, err)
	}

	if c, ok := bf().(HostConfigurer); ok {
		h.Configurer = c

		return nil
	}

	return errUnsupportedOS
}
