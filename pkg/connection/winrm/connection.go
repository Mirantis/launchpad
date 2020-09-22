package winrm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/masterzen/winrm"
	"github.com/packer-community/winrmcp/winrmcp"
)

// Connection describes a WinRM connection with its configuration options
type Connection struct {
	Address       string
	User          string
	Port          int
	Password      string
	UseHTTPS      bool
	Insecure      bool
	UseNTLM       bool
	CACertPath    string
	CertPath      string
	KeyPath       string
	TLSServerName string

	caCert  []byte
	key     []byte
	cert    []byte
	client  *winrm.Client
	winrmcp *winrmcp.Winrmcp
}

func (c *Connection) initWinrmcp() error {
	if c.winrmcp != nil {
		return nil
	}

	w, err := winrmcp.New(c.Address, &winrmcp.Config{
		Auth: winrmcp.Auth{
			User:     c.User,
			Password: c.Password,
		},
		Https:                 c.UseHTTPS,
		Insecure:              c.Insecure,
		TLSServerName:         c.TLSServerName,
		CACertBytes:           c.cert,
		MaxOperationsPerShell: 5,
	})

	if err != nil {
		return err
	}

	c.winrmcp = w
	return nil
}

// SetWindows is here to satisfy the interface, WinRM hosts are expected to always run windows
func (c *Connection) SetWindows(v bool) {
}

// IsWindows is here to satisfy the interface, WinRM hosts are expected to always run windows
func (c *Connection) IsWindows() bool {
	return true
}

func (c *Connection) loadCertificates() error {
	c.caCert = nil
	if c.CACertPath != "" {
		ca, err := ioutil.ReadFile(c.CACertPath)
		if err != nil {
			return err
		}
		c.caCert = ca
	}

	c.cert = nil
	if c.CertPath != "" {
		cert, err := ioutil.ReadFile(c.CertPath)
		if err != nil {
			return err
		}
		c.cert = cert
	}

	c.key = nil
	if c.KeyPath != "" {
		key, err := ioutil.ReadFile(c.KeyPath)
		if err != nil {
			return err
		}
		c.key = key
	}

	return nil
}

// Connect opens the WinRM connection
func (c *Connection) Connect() error {
	if err := c.loadCertificates(); err != nil {
		return fmt.Errorf("%s: failed to load certificates: %s", c.Address, err)
	}

	endpoint := &winrm.Endpoint{
		Host:          c.Address,
		Port:          c.Port,
		HTTPS:         c.UseHTTPS,
		Insecure:      c.Insecure,
		CACert:        c.caCert,
		Cert:          c.cert,
		Key:           c.key,
		TLSServerName: c.TLSServerName,
		Timeout:       60 * time.Second,
	}

	client, err := winrm.NewClient(endpoint, c.User, c.Password)
	if err != nil {
		return err
	}
	c.client = client

	_, err = client.CreateShell()

	if err != nil {
		return err
	}

	return nil
}

// Disconnect closes the WinRM connection
func (c *Connection) Disconnect() {
	c.client = nil
}

// ExecCmd executes a command on the host
func (c *Connection) ExecCmd(cmd string, stdin string, streamStdout bool, sensitiveCommand bool) error {
	shell, err := c.client.CreateShell()
	if err != nil {
		return err
	}

	if !sensitiveCommand {
		log.Debugf("%s: executing command: %s", c.Address, cmd)
	}

	command, err := shell.Execute(cmd)
	if err != nil {
		return err
	}

	if stdin != "" {
		if sensitiveCommand || len(stdin) > 256 {
			log.Debugf("%s: writing %d bytes to command stdin", c.Address, len(stdin))
		} else {
			log.Debugf("%s: writing %d bytes to command stdin: %s", c.Address, len(stdin), stdin)
		}

		go func() {
			command.Stdin.Write([]byte(stdin))
			command.Stdin.Close()
		}()
	} else {
		command.Stdin.Close()
	}

	multiReader := io.MultiReader(command.Stdout, command.Stderr)
	outputScanner := bufio.NewScanner(multiReader)

	for outputScanner.Scan() {
		if streamStdout {
			log.Infof("%s:  %s", c.Address, outputScanner.Text())
		} else {
			log.Debugf("%s:  %s", c.Address, outputScanner.Text())
		}
	}
	if err := outputScanner.Err(); err != nil {
		log.Errorf("%s:  %s", c.Address, err.Error())
	}

	command.Wait()
	command.Close()
	shell.Close()

	if command.ExitCode() > 0 {
		return fmt.Errorf("%s: command failed", c.Address)
	}

	return nil
}

// ExecWithOutput executes a command on the host and returns its output
func (c *Connection) ExecWithOutput(cmd string) (string, error) {
	shell, err := c.client.CreateShell()
	if err != nil {
		return "", err
	}

	command, err := shell.Execute(cmd)
	if err != nil {
		return "", err
	}

	var outWriter, errWriter bytes.Buffer
	go io.Copy(&outWriter, command.Stdout)
	go io.Copy(&errWriter, command.Stderr)

	command.Wait()
	shell.Close()

	if command.ExitCode() > 0 {
		return "", fmt.Errorf("%s: command failed (%d): %s", c.Address, command.ExitCode(), errWriter.String())
	}

	return strings.TrimSpace(outWriter.String()), nil
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (c *Connection) WriteFileLarge(src, dst string) error {
	if err := c.initWinrmcp(); err != nil {
		return err
	}
	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	log.Infof("%s: copying %d bytes to %s", c.Address, stat.Size(), dst)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	return c.winrmcp.Write(dst, in)
}
