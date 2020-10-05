package winrm

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"
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

	// if c.UseHTTPS {
	// 	params.TransportDecorator = func() winrm.Transporter { return &winrm.ClientAuthRequest{} }
	// }

	client, err := winrm.NewClientWithParameters(endpoint, c.User, c.Password, params)

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
	shell := lease.shell

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

	go func() {
		defer wg.Done()
		outputScanner := bufio.NewScanner(command.Stderr)

		for outputScanner.Scan() {
			o.AddOutput(c.Address, outputScanner.Text()+"\n")
		}

		if err := outputScanner.Err(); err != nil {
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
	lease.Release()

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

// from jbrekelmans/go-winrm/util.go PowerShellSingleQuotedStringLiteral
func Escape(v string) string {
	var sb strings.Builder
	_, _ = sb.WriteRune('\'')
	for _, rune := range v {
		var esc string
		switch rune {
		case '\n':
			esc = "`n"
		case '\r':
			esc = "`r"
		case '\t':
			esc = "`t"
		case '\a':
			esc = "`a"
		case '\b':
			esc = "`b"
		case '\f':
			esc = "`f"
		case '\v':
			esc = "`v"
		case '"':
			esc = "`\""
		case '\'':
			esc = "`'"
		case '`':
			esc = "``"
		case '\x00':
			esc = "`0"
		default:
			_, _ = sb.WriteRune(rune)
			continue
		}
		_, _ = sb.WriteString(esc)
	}
	_, _ = sb.WriteRune('\'')
	return sb.String()
}

func PSEncode(psCmd string) string {
	// 2 byte chars to make PowerShell happy
	wideCmd := ""
	for _, b := range []byte(psCmd) {
		wideCmd += string(b) + "\x00"
	}

	// Base64 encode the command
	input := []uint8(wideCmd)
	return base64.StdEncoding.EncodeToString(input)
}

// Powershell wraps a PowerShell script
// and prepares it for execution by the winrm client
func Powershell(psCmd string) string {
	encodedCmd := PSEncode(psCmd)

	log.Debugf("encoded powershell command: %s", psCmd)
	// Create the powershell.exe command line to execute the script
	return fmt.Sprintf("powershell.exe -NonInteractive -NoProfile -EncodedCommand %s", encodedCmd)
}
