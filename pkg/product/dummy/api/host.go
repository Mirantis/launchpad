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
	"github.com/creasty/defaults"

	log "github.com/sirupsen/logrus"
)

// Host contains all the needed details to work with hosts
type Host struct {
	Address   string        `yaml:"address" validate:"required,hostname|ip"`
	WinRM     *common.WinRM `yaml:"winRM,omitempty"`
	SSH       *common.SSH   `yaml:"ssh,omitempty"`
	Localhost bool          `yaml:"localhost,omitempty"`

	Connection connection.Connection `yaml:"-"`

	name string
}

func (h *Host) generateName() string {
	if h.Localhost {
		return "localhost"
	}

	if h.WinRM != nil {
		return fmt.Sprintf("%s:%d", h.Address, h.WinRM.Port)
	}

	if h.SSH != nil {
		return fmt.Sprintf("%s:%d", h.Address, h.SSH.Port)
	}

	return h.Address
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
		h.SSH = common.DefaultSSH()
	}

	return nil
}

// Connect to the host
// TODO in some sort of generic Connectable
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
