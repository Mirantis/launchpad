package winrm

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/masterzen/winrm"
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

	caCert []byte
	key    []byte
	cert   []byte
	client *winrm.Client
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
		sb := bytes.NewBufferString(stdin)
		log.Debugf("%s: writing data to command stdin: %s", c.Address, stdin)
		go io.Copy(command.Stdin, sb)
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

	return outWriter.String(), nil
}
