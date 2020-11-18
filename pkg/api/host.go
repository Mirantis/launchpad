package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/connection"
	"github.com/Mirantis/mcc/pkg/connection/local"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/util"
	retry "github.com/avast/retry-go"
	"github.com/creasty/defaults"

	log "github.com/sirupsen/logrus"
)

// OsRelease host operating system info
type OsRelease struct {
	ID      string
	IDLike  string
	Name    string
	Version string
}

// HostMetadata resolved metadata for host
type HostMetadata struct {
	Hostname            string
	LongHostname        string
	InternalAddress     string
	EngineVersion       string
	Os                  *OsRelease
	EngineInstallScript string
	ImagesToUpload      []string
	TotalImageBytes     uint64
}

// MSRMetadata is metadata needed by MSR for configuration and is gathered at
// the GatherFacts phase and at the end of each configuration phase
type MSRMetadata struct {
	Installed               bool
	InstalledVersion        string
	InstalledBootstrapImage string
	ReplicaID               string
}

type errors struct {
	errors []string
}

func (errors *errors) Count() int {
	return len(errors.errors)
}

func (errors *errors) Add(e string) {
	errors.errors = append(errors.errors, e)
}

func (errors *errors) Addf(template string, args ...interface{}) {
	errors.errors = append(errors.errors, fmt.Sprintf(template, args...))
}

func (errors *errors) String() string {
	if errors.Count() == 0 {
		return ""
	}

	return "- " + strings.Join(errors.errors, "\n- ")
}

// BeforeAfter is the a child struct for the Hooks struct, containing sections for Before and After
type BeforeAfter struct {
	Before *[]string `yaml:"before" default:"[]"`
	After  *[]string `yaml:"after" default:"[]"`
}

// Hooks is a list of hook-points
type Hooks struct {
	Apply *BeforeAfter `yaml:"apply" default:"{}"`
	Reset *BeforeAfter `yaml:"reset" default:"{}"`
}

// Host contains all the needed details to work with hosts
type Host struct {
	Address          string            `yaml:"address" validate:"required,hostname|ip"`
	Role             string            `yaml:"role" validate:"oneof=manager worker msr"`
	PrivateInterface string            `yaml:"privateInterface,omitempty" validate:"omitempty,gt=2"`
	DaemonConfig     GenericHash       `yaml:"engineConfig,flow,omitempty" default:"{}"`
	Environment      map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks            *Hooks            `yaml:"hooks,omitempty" default:"{}"`
	ImageDir         string            `yaml:"imageDir,omitempty"`

	WinRM     *WinRM `yaml:"winRM,omitempty"`
	SSH       *SSH   `yaml:"ssh,omitempty"`
	Localhost bool   `yaml:"localhost,omitempty"`

	Metadata    *HostMetadata  `yaml:"-"`
	MSRMetadata *MSRMetadata   `yaml:"-"`
	Configurer  HostConfigurer `yaml:"-"`
	Errors      errors         `yaml:"-"`

	Connection connection.Connection `yaml:"-"`

	name string
}

func (h *Host) generateName() string {
	var role string

	switch h.Role {
	case "manager":
		role = "M"
	case "worker":
		role = "W"
	case "msr":
		role = "R"
	}

	if h.Localhost {
		return fmt.Sprintf("%s localhost", role)
	}

	if h.WinRM != nil {
		return fmt.Sprintf("%s %s:%d", role, h.Address, h.WinRM.Port)
	}

	if h.SSH != nil {
		return fmt.Sprintf("%s %s:%d", role, h.Address, h.SSH.Port)
	}

	return fmt.Sprintf("%s %s", role, h.Address) // I don't think it should go here except in tests
}

// String returns a name / string identifier for the host
func (h *Host) String() string {
	if h.name == "" {
		h.name = h.generateName()
	}
	return h.name
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(h)

	type host Host
	yh := (*host)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	if h.WinRM == nil && h.SSH == nil && !h.Localhost {
		h.SSH = DefaultSSH()
	}

	return nil
}

// Connect to the host
func (h *Host) Connect() error {
	var c connection.Connection

	var proto string

	if h.Localhost {
		c = local.NewConnection()
		proto = "Local"
	} else if h.WinRM == nil {
		c = h.SSH.NewConnection(h.Address)
		proto = "SSH"
	} else {
		c = h.WinRM.NewConnection(h.Address)
		proto = "WinRM"
	}

	c.SetName(h.String())

	log.Infof("%s: opening %s connection", h, proto)
	if err := c.Connect(); err != nil {
		h.Connection = nil
		return err
	}

	log.Infof("%s: %s connection opened", h, proto)

	h.Connection = c

	return nil
}

// Disconnect the host
func (h *Host) Disconnect() {
	if h.Connection != nil {
		h.Connection.Disconnect()
	}
	h.Connection = nil
}

// Exec a command on the host
func (h *Host) Exec(cmd string, opts ...exec.Option) error {
	return h.Connection.Exec(cmd, opts...)
}

// ExecWithOutput execs a command on the host and returns output
func (h *Host) ExecWithOutput(cmd string, opts ...exec.Option) (string, error) {
	var output string
	opts = append(opts, exec.Output(&output))
	err := h.Exec(cmd, opts...)
	return strings.TrimSpace(output), err
}

// ExecAll execs a slice of commands on the host
func (h *Host) ExecAll(cmds []string) error {
	for _, cmd := range cmds {
		log.Infof("%s: Executing: %s", h, cmd)
		output, err := h.ExecWithOutput(cmd)
		if err != nil {
			log.Errorf("%s: %s", h, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h)))
			return err
		}
		if strings.TrimSpace(output) != "" {
			log.Infof("%s: %s", h, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h)))
		}
	}
	return nil
}

func trimOutput(output []byte) string {
	if len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// AuthenticateDocker performs a docker login on the host using local REGISTRY_USERNAME
// and REGISTRY_PASSWORD when set
func (h *Host) AuthenticateDocker(imageRepo string) error {
	if user := os.Getenv("REGISTRY_USERNAME"); user != "" {
		pass := os.Getenv("REGISTRY_PASSWORD")
		if pass == "" {
			return fmt.Errorf("REGISTRY_PASSWORD not set")
		}

		log.Infof("%s: authenticating docker for image repo %s", h, imageRepo)
		if strings.HasPrefix(imageRepo, "docker.io/") { // docker.io is a special case for auth
			imageRepo = ""
		}
		return h.Configurer.AuthenticateDocker(user, pass, imageRepo)
	}
	return nil
}

// SwarmAddress determines the swarm address for the host
func (h *Host) SwarmAddress() string {
	return fmt.Sprintf("%s:%d", h.Metadata.InternalAddress, 2377)
}

// IsWindows returns true if host has been detected running windows
func (h *Host) IsWindows() bool {
	if h.Metadata == nil {
		return false
	}
	if h.Metadata.Os == nil {
		return false
	}
	return strings.HasPrefix(h.Metadata.Os.ID, "windows-")
}

// EngineVersion returns the current engine version installed on the host
func (h *Host) EngineVersion() (string, error) {
	version, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`version -f "{{.Server.Version}}"`))
	if err != nil {
		return "", fmt.Errorf("failed to get docker engine version: %s", err.Error())
	}

	return version, nil
}

// CheckHTTPStatus will perform a web request to the url and return an error if the http status is not the expected
func (h *Host) CheckHTTPStatus(url string, expected int) error {
	status, err := h.Configurer.HTTPStatus(url)
	if err != nil {
		return err
	}

	log.Debugf("%s: response code: %d, expected %d", h, status, expected)
	if status != expected {
		return fmt.Errorf("unexpected response code %d", status)
	}

	return nil
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (h *Host) WriteFileLarge(src, dst string) error {
	startTime := time.Now()
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	size := stat.Size()

	log.Infof("%s: uploading %s to %s", h, util.FormatBytes(uint64(stat.Size())), dst)

	if err := h.Connection.Upload(src, dst); err != nil {
		return fmt.Errorf("upload failed: %s", err.Error())
	}

	duration := time.Since(startTime).Seconds()
	speed := float64(size) / duration
	log.Infof("%s: transfered %s in %.1f seconds (%s/s)", h, util.FormatBytes(uint64(size)), duration, util.FormatBytes(uint64(speed)))

	return nil
}

// Reboot reboots the host and waits for it to become responsive
func (h *Host) Reboot() error {
	log.Infof("%s: rebooting", h.Address)
	h.Exec(h.Configurer.RebootCommand())
	log.Infof("%s: waiting for host to go offline", h.Address)
	if err := h.waitForHost(false); err != nil {
		return err
	}
	h.Disconnect()

	log.Infof("%s: waiting for reconnection", h.Address)
	err := retry.Do(
		func() error {
			return h.Connect()
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
	)

	if err != nil {
		return fmt.Errorf("unable to reconnect after reboot")
	}

	log.Infof("%s: waiting for host to become active", h.Address)
	if err := h.waitForHost(true); err != nil {
		return err
	}

	if err != nil {
		return fmt.Errorf("unable to reconnect after reboot")
	}

	return nil
}

// when state is true wait for host to become active, when state is false, wait for connection to go down
func (h *Host) waitForHost(state bool) error {
	err := retry.Do(
		func() error {
			err := h.Exec("echo")
			if !state && err == nil {
				return fmt.Errorf("still online")
			} else if state && err != nil {
				return fmt.Errorf("still offline")
			}
			return nil
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
	)
	if err != nil {
		return fmt.Errorf("failed to wait for host to go offline")
	}
	return nil
}
