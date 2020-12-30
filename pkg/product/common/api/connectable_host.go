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
	log "github.com/sirupsen/logrus"
)

type ConnectableHost struct {
	Address    string                `yaml:"address" validate:"required,hostname|ip"`
	WinRM      *WinRM                `yaml:"winRM,omitempty"`
	SSH        *SSH                  `yaml:"ssh,omitempty"`
	Localhost  bool                  `yaml:"localhost,omitempty"`
	Connection connection.Connection `yaml:"-"`

	name string
}

// Exec a command on the host
func (h *ConnectableHost) Exec(cmd string, opts ...exec.Option) error {
	return h.Connection.Exec(cmd, opts...)
}

// ExecWithOutput execs a command on the host and returns output
func (h *ConnectableHost) ExecWithOutput(cmd string, opts ...exec.Option) (string, error) {
	var output string
	opts = append(opts, exec.Output(&output))
	err := h.Exec(cmd, opts...)
	return strings.TrimSpace(output), err
}

// ExecAll execs a slice of commands on the host
func (h *ConnectableHost) ExecAll(cmds []string) error {
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

func (h *ConnectableHost) String() string {
	if h.name == "" {
		h.name = h.generateName()
	}
	return h.name
}

func (h *ConnectableHost) generateName() string {
	if h.Localhost {
		return fmt.Sprintf("localhost")
	}

	if h.WinRM != nil {
		return fmt.Sprintf("%s:%d", h.Address, h.WinRM.Port)
	}

	if h.SSH != nil {
		return fmt.Sprintf("%s:%d", h.Address, h.SSH.Port)
	}

	return fmt.Sprintf("%s", h.Address) // I don't think it should go here except in tests
}

// Connect to the host
func (h *ConnectableHost) Connect() error {
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
func (h *ConnectableHost) Disconnect() {
	if h.Connection != nil {
		h.Connection.Disconnect()
	}
	h.Connection = nil
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (h *ConnectableHost) WriteFileLarge(src, dst string) error {
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
