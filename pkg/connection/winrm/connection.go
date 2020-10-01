package winrm

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Mirantis/mcc/pkg/exec"

	"github.com/Azure/go-ntlmssp"
	"github.com/jbrekelmans/winrm"
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

	shellpool *ShellPool
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

	tlsConfig := &tls.Config{
		ServerName:         c.TLSServerName,
		InsecureSkipVerify: c.Insecure,
	}

	if len(c.caCert) > 0 {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(c.caCert)
	}

	if len(c.cert) > 0 {
		cert, err := tls.LoadX509KeyPair(string(c.cert), string(c.key))
		if err != nil {
			return err
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	var tlsTransport *http.Transport
	if c.UseHTTPS {
		tlsTransport = &http.Transport{
			MaxConnsPerHost: 300,
			TLSClientConfig: tlsConfig,
		}
	}

	httpClient := &http.Client{}

	if c.UseNTLM {
		httpClient.Transport = &ntlmssp.Negotiator{
			RoundTripper: tlsTransport,
		}
	} else {
		httpClient.Transport = tlsTransport
	}

	maxEnvelopeSize := 500 * 1000
	client, err := winrm.NewClient(context.Background(), c.UseHTTPS, c.Address, c.Port, c.User, c.Password, httpClient, &maxEnvelopeSize)
	if err != nil {
		return err
	}

	c.client = client
	c.shellpool = NewShellPool(client)

	return nil
}

// Disconnect closes the WinRM connection
func (c *Connection) Disconnect() {
	c.client = nil
	c.shellpool = nil
}

// Exec executes a command on the host
func (c *Connection) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	lease := c.shellpool.Get()
	if lease == nil {
		return fmt.Errorf("%s: failed to create a shell", c.Address)
	}
	defer lease.Release()
	shell := lease.shell

	o.LogCmd(c.Address, cmd)

	command, err := shell.StartCommand(cmd, nil, false, false)
	if err != nil {
		return err
	}
	defer command.Signal()

	if o.Stdin != "" {
		o.LogStdin(c.Address)

		go func() { command.SendInput([]byte(o.Stdin), true) }()
	}

	multiReader := io.MultiReader(command.Stdout, command.Stderr)
	outputScanner := bufio.NewScanner(multiReader)

	go func() {
		for outputScanner.Scan() {
			o.AddOutput(c.Address, outputScanner.Text()+"\n")
		}

		if err := outputScanner.Err(); err != nil {
			o.LogErrorf("%s:  %s", c.Address, err.Error())
		}
	}()

	command.Wait()

	if command.ExitCode() > 0 {
		return fmt.Errorf("%s: command failed", c.Address)
	}

	return nil
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (c *Connection) WriteFileLarge(src, dst string) error {
	return Upload(src, dst, c)
}
