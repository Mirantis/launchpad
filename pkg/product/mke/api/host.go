package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	retry "github.com/avast/retry-go"
	"github.com/creasty/defaults"

	log "github.com/sirupsen/logrus"
)

// HostMetadata resolved metadata for host
type HostMetadata struct {
	Hostname           string
	LongHostname       string
	InternalAddress    string
	MCRVersion         string
	MCRInstallScript   string
	MCRRestartRequired bool
	Os                 *common.OsRelease
	ImagesToUpload     []string
	TotalImageBytes    uint64
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

// Host contains all the needed details to work with hosts
type Host struct {
	Role             string             `yaml:"role" validate:"oneof=manager worker msr"`
	PrivateInterface string             `yaml:"privateInterface,omitempty" validate:"omitempty,gt=2"`
	DaemonConfig     common.GenericHash `yaml:"mcrConfig,flow,omitempty" default:"{}"`
	Environment      map[string]string  `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks            common.Hooks       `yaml:"hooks,omitempty" validate:"dive,keys,oneof=apply reset,endkeys,dive,keys,oneof=before after,endkeys,omitempty"`
	ImageDir         string             `yaml:"imageDir,omitempty"`

	Metadata    *HostMetadata  `yaml:"-"`
	MSRMetadata *MSRMetadata   `yaml:"-"`
	Configurer  HostConfigurer `yaml:"-"`
	Errors      errors         `yaml:"-"`

	common.ConnectableHost `yaml:",inline"`

	name string
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	defaults.Set(h)
	defaults.Set(h.ConnectableHost)

	type host Host
	yh := (*host)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	if h.WinRM == nil && h.SSH == nil && !h.Localhost {
		h.SSH = common.DefaultSSH()
	}

	return nil
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

// MCRVersion returns the current engine version installed on the host
func (h *Host) MCRVersion() (string, error) {
	version, err := h.ExecWithOutput(h.Configurer.DockerCommandf(`version -f "{{.Server.Version}}"`))
	if err != nil {
		return "", fmt.Errorf("failed to get container runtime version: %s", err.Error())
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

// ConfigureMCR writes the docker engine daemon.json and toggles the host Metadata MCRRestartRequired flag if changed
func (h *Host) ConfigureMCR() error {
	if len(h.DaemonConfig) > 0 {
		daemonJSONData, err := json.Marshal(h.DaemonConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal daemon json config: %w", err)
		}
		daemonJSON := string(daemonJSONData)

		cfg := h.Configurer.MCRConfigPath()
		oldConfig := ""
		if h.Configurer.FileExist(cfg) {
			f, err := h.Configurer.ReadFile(cfg)
			if err != nil {
				return err
			}
			oldConfig = f

			log.Debugf("deleting %s", cfg)
			if err := h.Configurer.DeleteFile(cfg); err != nil {
				return err
			}
		}

		log.Debugf("writing %s", cfg)
		if err := h.Configurer.WriteFile(cfg, daemonJSON, "0700"); err != nil {
			return err
		}
		if h.Metadata.MCRVersion != "" && oldConfig != daemonJSON {
			h.Metadata.MCRRestartRequired = true
		}
	}
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
