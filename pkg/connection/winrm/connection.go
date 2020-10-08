package winrm

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/Mirantis/mcc/pkg/exec"
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
		TLSServerName: c.TLSServerName,
		Timeout:       60 * time.Second,
	}

	if len(c.caCert) > 0 {
		endpoint.CACert = c.caCert
	}

	if len(c.cert) > 0 {
		endpoint.Cert = c.cert
	}

	if len(c.key) > 0 {
		endpoint.Key = c.key
	}

	params := winrm.DefaultParameters

	if c.UseNTLM {
		params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientNTLM{} }
	}

	if c.UseHTTPS && len(c.cert) > 0 {
		params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientAuthRequest{} }
	}

	client, err := winrm.NewClientWithParameters(endpoint, c.User, c.Password, params)

	if err != nil {
		return err
	}

	c.client = client

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
	defer shell.Close()

	o.LogCmd(c.Address, cmd)

	var wg sync.WaitGroup
	wg.Add(2)

	command, err := shell.Execute(cmd)
	if err != nil {
		return err
	}

	if o.Stdin != "" {
		o.LogStdin(c.Address)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer command.Stdin.Close()
			_, err := command.Stdin.Write([]byte(o.Stdin))
			if err != nil {
				log.Errorf("failed to send command stdin: %s", err.Error())
			}
			log.Tracef("%s: input loop exited", c.Address)
		}()
	}

	go func() {
		defer wg.Done()
		outputScanner := bufio.NewScanner(command.Stdout)

		for outputScanner.Scan() {
			o.AddOutput(c.Address, outputScanner.Text()+"\n")
		}

		if err := outputScanner.Err(); err != nil {
			o.LogErrorf("%s:  %s", c.Address, err.Error())
		}
		command.Stdout.Close()
		log.Tracef("%s: stdout loop exited", c.Address)
	}()

	gotErrors := false

	go func() {
		defer wg.Done()
		outputScanner := bufio.NewScanner(command.Stderr)

		for outputScanner.Scan() {
			gotErrors = true
			o.AddOutput(c.Address+" (stderr)", outputScanner.Text()+"\n")
		}

		if err := outputScanner.Err(); err != nil {
			gotErrors = true
			o.LogErrorf("%s:  %s", c.Address, err.Error())
		}
		command.Stdout.Close()
		log.Tracef("%s: stderr loop exited", c.Address)
	}()

	log.Tracef("%s: waiting for command exit", c.Address)

	command.Wait()
	log.Tracef("%s: command exited", c.Address)

	log.Tracef("%s: waiting for syncgroup done", c.Address)
	wg.Wait()
	log.Tracef("%s: syncgroup done", c.Address)

	err = command.Close()
	if err != nil {
		log.Warnf("%s: %s", c.Address, err.Error())
	}

	if command.ExitCode() > 0 || gotErrors {
		return fmt.Errorf("command failed")
	}

	return nil
}
