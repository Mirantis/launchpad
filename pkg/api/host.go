package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/Mirantis/mcc/pkg/connection"
	"github.com/Mirantis/mcc/pkg/connection/local"
	"github.com/Mirantis/mcc/pkg/exec"
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
	Role             string            `yaml:"role" validate:"oneof=manager worker dtr"`
	PrivateInterface string            `yaml:"privateInterface,omitempty" validate:"omitempty,gt=2"`
	DaemonConfig     GenericHash       `yaml:"engineConfig,flow,omitempty" default:"{}"`
	Environment      map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks            *Hooks            `yaml:"hooks,omitempty" default:"{}"`

	WinRM     *WinRM `yaml:"winRM,omitempty"`
	SSH       *SSH   `yaml:"ssh,omitempty"`
	Localhost bool   `yaml:"localhost,omitempty"`

	Metadata   *HostMetadata  `yaml:"-"`
	Configurer HostConfigurer `yaml:"-"`
	Errors     errors         `yaml:"-"`

	Connection connection.Connection `yaml:"-"`
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

	if h.Localhost {
		c = local.NewConnection()
	} else if h.WinRM == nil {
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
		log.Infof("%s: Executing: %s", h.Address, cmd)
		output, err := h.ExecWithOutput(cmd)
		if err != nil {
			log.Errorf("%s: %s", h.Address, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h.Address)))
			return err
		}
		if strings.TrimSpace(output) != "" {
			log.Infof("%s: %s", h.Address, strings.ReplaceAll(output, "\n", fmt.Sprintf("\n%s: ", h.Address)))
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

// CheckHTTPStatus will perform a web request to the url and return an error if the http status is not the expected
func (h *Host) CheckHTTPStatus(url string, expected int) error {
	status, err := h.Configurer.HTTPStatus(url)
	if err != nil {
		return err
	}

	log.Debugf("%s: response code: %d, expected %d", h.Address, status, expected)
	if status != expected {
		return fmt.Errorf("%s: unexpected response code %d", h.Address, status)
	}

	return nil
}
