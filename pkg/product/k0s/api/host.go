package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/connection"
	"github.com/Mirantis/mcc/pkg/connection/local"
	"github.com/Mirantis/mcc/pkg/exec"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
	// "github.com/prometheus/common/log"
)

// Host contains all the needed details to work with hosts
type Host struct {
	Address      string                `yaml:"address" validate:"required,hostname|ip"`
	WinRM        *common.WinRM         `yaml:"winRM,omitempty"`
	SSH          *common.SSH           `yaml:"ssh,omitempty"`
	Localhost    bool                  `yaml:"localhost,omitempty"`
	Role         string                `yaml:"role" validate:"oneof=server worker"`
	UploadBinary bool                  `yaml:"uploadBinary,omitempty"`
	K0sBinary    string                `yaml:"k0sBinary,omitempty" validate:"file"`
	Connection   connection.Connection `yaml:"-"`
	Configurer   HostConfigurer        `yaml:"-"`
	Metadata     *HostMetadata         `yaml:"-"`

	name string
}

// HostMetadata resolved metadata for host
type HostMetadata struct {
	Hostname        string
	LongHostname    string
	InternalAddress string
	K0sVersion      string
	Os              *common.OsRelease
	JoinToken       string
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

// K0sVersion returns installed version of k0s
func (h *Host) K0sVersion() (string, error) {
	return h.ExecWithOutput("k0s version")
}

func (h *Host) String() string {
	if h.name == "" {
		h.name = h.generateName()
	}
	return h.name
}

func (h *Host) generateName() string {
	var role string

	switch h.Role {
	case "server":
		role = "S"
	case "worker":
		role = "W"
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

	c.SetName(h.String())

	// log.Infof("%s: opening %s connection", h, proto)
	if err := c.Connect(); err != nil {
		h.Connection = nil
		return err
	}

	// log.Infof("%s: %s connection opened", h, proto)

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

//PrepareConfig persists k0s.yaml config to the host
func (h *Host) PrepareConfig(config *common.GenericHash) error {
	output, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return h.Configurer.WriteFile(h.Configurer.K0sConfigPath(), string(output), "0700")
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
