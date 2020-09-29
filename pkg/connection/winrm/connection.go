package winrm

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/Mirantis/mcc/pkg/exec"

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

// Exec executes a command on the host
func (c *Connection) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	shell, err := c.client.CreateShell()
	if err != nil {
		return err
	}

	o.LogCmd(c.Address, cmd)

	command, err := shell.Execute(cmd)
	if err != nil {
		return err
	}

	if o.Stdin != "" {
		o.LogStdin(c.Address)

		go func() {
			command.Stdin.Write([]byte(o.Stdin))
			command.Stdin.Close()
		}()
	} else {
		command.Stdin.Close()
	}

	multiReader := io.MultiReader(command.Stdout, command.Stderr)
	outputScanner := bufio.NewScanner(multiReader)

	for outputScanner.Scan() {
		o.AddOutput(c.Address, outputScanner.Text()+"\n")
	}

	if err := outputScanner.Err(); err != nil {
		o.LogErrorf("%s:  %s", c.Address, err.Error())
	}

	command.Wait()
	command.Close()
	shell.Close()

	if command.ExitCode() > 0 {
		return fmt.Errorf("%s: command failed", c.Address)
	}

	return nil
}
