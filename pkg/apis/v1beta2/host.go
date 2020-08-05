package v1beta2

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/connection"
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
	EngineInstallScript *string
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

// Host contains all the needed details to work with hosts
type Host struct {
	Address          string            `yaml:"address" validate:"required,hostname|ip"`
	Role             string            `yaml:"role" validate:"oneof=manager worker dtr"`
	PrivateInterface string            `yaml:"privateInterface,omitempty" default:"eth0" validate:"gt=2"`
	DaemonConfig     GenericHash       `yaml:"engineConfig,flow" default:"{}"`
	Environment      map[string]string `yaml:"environment,flow,omitempty" default:"{}"`

	WinRM *WinRM `yaml:"winRM,omitempty"`
	SSH   *SSH   `yaml:"ssh,omitempty"`

	Metadata   *HostMetadata  `yaml:"-"`
	Configurer HostConfigurer `yaml:"-"`
	Errors     errors         `yaml:"-"`

	Connection connection.Connection `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(h)

	type yhost Host
	yh := (*yhost)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	if h.WinRM == nil && h.SSH == nil {
		h.SSH = DefaultSSH()
	}

	return nil
}

// Connect to the host
func (h *Host) Connect() error {
	var c connection.Connection

	if h.WinRM == nil {
		c = h.SSH.NewConnection(h.Address)
	} else {
		c = h.WinRM.NewConnection(h.Address)
	}

	if err := c.Connect(); err != nil {
		h.Connection = nil
		return err
	}

	h.Connection = c

	return nil
}

// Disconnect the host
func (h *Host) Disconnect() {
	if h.Connection != nil {
		h.Connection.Disconnect()
	}
}

// ExecCmd a command on the host piping stdin and streams the logs
func (h *Host) ExecCmd(cmd string, stdin string, streamStdout bool, sensitiveCommand bool) error {
	return h.Connection.ExecCmd(cmd, stdin, streamStdout, sensitiveCommand)
}

// Exec a command on the host and streams the logs
func (h *Host) Exec(cmd string) error {
	return h.ExecCmd(cmd, "", false, false)
}

// Execf a printf-formatted command on the host and streams the logs
func (h *Host) Execf(cmd string, args ...interface{}) error {
	return h.Exec(fmt.Sprintf(cmd, args...))
}

// ExecWithOutput execs a command on the host and return output
func (h *Host) ExecWithOutput(cmd string) (string, error) {
	return h.Connection.ExecWithOutput(cmd)
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
			return fmt.Errorf("%s: REGISTRY_PASSWORD not set", h.Address)
		}

		log.Infof("%s: authenticating docker for image repo %s", h.Address, imageRepo)
		if strings.HasPrefix(imageRepo, "docker.io/") { // docker.io is a special case for auth
			imageRepo = ""
		}
		return h.Configurer.AuthenticateDocker(user, pass, imageRepo)
	}
	log.Debugf("%s: REGISTRY_USERNAME not set, not authenticating", h.Address)
	return nil
}

// PullImage pulls the named docker image on the host
func (h *Host) PullImage(name string) error {
	output, err := h.ExecWithOutput(h.Configurer.DockerCommandf("pull %s", name))
	if err != nil {
		log.Warnf("%s: failed to pull image %s: \n%s", h.Address, name, output)
		return err
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
func (h *Host) EngineVersion() string {
	version, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`version -f "{{.Server.Version}}"`))
	if err != nil {
		log.Debugf("%s: failed to get docker engine version: %s: %s", h.Address, version, err)
		return ""
	}

	if version == "" {
		log.Infof("%s: docker engine not installed", h.Address)
	} else {
		log.Infof("%s: is running docker engine version %s", h.Address, h.Metadata.EngineVersion)
	}

	return version
}
