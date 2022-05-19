package api

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
	retry "github.com/avast/retry-go"
	"github.com/creasty/defaults"
	"github.com/k0sproject/dig"
	"github.com/k0sproject/rig"
	"github.com/k0sproject/rig/os/registry"

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
	rig.Connection `yaml:",inline"`

	Role             string            `yaml:"role" validate:"oneof=manager worker msr"`
	PrivateInterface string            `yaml:"privateInterface,omitempty" validate:"omitempty,gt=2"`
	DaemonConfig     dig.Mapping       `yaml:"mcrConfig,flow,omitempty" default:"{}"`
	Environment      map[string]string `yaml:"environment,flow,omitempty" default:"{}"`
	Hooks            common.Hooks      `yaml:"hooks,omitempty" validate:"dive,keys,oneof=apply reset,endkeys,dive,keys,oneof=before after,endkeys,omitempty"`
	ImageDir         string            `yaml:"imageDir,omitempty"`

	Metadata    *HostMetadata  `yaml:"-"`
	MSRMetadata *MSRMetadata   `yaml:"-"`
	Configurer  HostConfigurer `yaml:"-"`
	Errors      errors         `yaml:"-"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (h *Host) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type host Host
	yh := (*host)(h)

	if err := unmarshal(yh); err != nil {
		return err
	}

	return defaults.Set(h)
}

// IsLocal returns true for localhost connections
func (h *Host) IsLocal() bool {
	return h.Protocol() == "Local"
}

// ExecAll execs a slice of commands on the host
func (h *Host) ExecAll(cmds []string) error {
	for _, cmd := range cmds {
		log.Infof("%s: Executing: %s", h, cmd)
		output, err := h.ExecOutput(cmd)
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
		return h.Configurer.AuthenticateDocker(h, user, pass, imageRepo)
	}
	return nil
}

// SwarmAddress determines the swarm address for the host
func (h *Host) SwarmAddress() string {
	return fmt.Sprintf("%s:%d", h.Metadata.InternalAddress, 2377)
}

// MCRVersion returns the current engine version installed on the host
func (h *Host) MCRVersion() (string, error) {
	version, err := h.ExecOutput(h.Configurer.DockerCommandf(`version -f "{{.Server.Version}}"`))
	if err != nil {
		return "", fmt.Errorf("failed to get container runtime version: %s", err.Error())
	}

	return version, nil
}

// CheckHTTPStatus will perform a web request to the url and return an error if the http status is not the expected
func (h *Host) CheckHTTPStatus(url string, expected int) error {
	status, err := h.Configurer.HTTPStatus(h, url)
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

// Reconnect disconnects and reconnects the host's connection
func (h *Host) Reconnect() error {
	h.Disconnect()

	log.Infof("%s: waiting for reconnection", h)
	return retry.Do(
		func() error {
			return h.Connect()
		},
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(60),
	)
}

// Reboot reboots the host and waits for it to become responsive
func (h *Host) Reboot() error {
	log.Infof("%s: rebooting", h)
	if err := h.Configurer.Reboot(h); err != nil {
		return err
	}
	log.Infof("%s: waiting for host to go offline", h)
	if err := h.waitForHost(false); err != nil {
		return err
	}
	h.Disconnect()

	log.Infof("%s: waiting for reconnection", h)
	if err := h.Reconnect(); err != nil {
		return fmt.Errorf("unable to reconnect after reboot")
	}

	log.Infof("%s: waiting for host to become active", h)
	if err := h.waitForHost(true); err != nil {
		return err
	}

	if err := h.Reconnect(); err != nil {
		return fmt.Errorf("unable to reconnect after reboot: %s", err.Error())
	}

	return nil
}

// ConfigureMCR writes the docker engine daemon.json and toggles the host Metadata MCRRestartRequired flag if changed
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
		log.Debugf("%s: failed to delete exising daemon.json: %s", h, err)
	}

	if err := h.Configurer.WriteFile(h, cfgPath, daemonJSONContent, "0600"); err != nil {
		return err
	}

	if h.Metadata.MCRVersion != "" {
		h.Metadata.MCRRestartRequired = true
		log.Debugf("%s: host marked for mcr restart", h)
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

// ResolveConfigurer assigns a rig-style configurer to the Host (see configurer/)
func (h *Host) ResolveConfigurer() error {
	bf, err := registry.GetOSModuleBuilder(*h.OSVersion)
	if err != nil {
		return err
	}

	if c, ok := bf().(HostConfigurer); ok {
		h.Configurer = c

		return nil
	}

	return fmt.Errorf("unsupported OS")
}
