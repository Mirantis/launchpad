package ssh

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	ssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/acarl005/stripansi"
	log "github.com/sirupsen/logrus"
)

// Connection describes an SSH connection
type Connection struct {
	Address string
	User    string
	Port    int
	KeyPath string

	isWindows bool
	knowOs    bool
	client    *ssh.Client

	done chan bool
}

// Disconnect closes the SSH connection
func (c *Connection) Disconnect() {
	c.client.Close()
	c.done <- true
}

// SetWindows can be used to tell the SSH connection to consider the host to be running Windows
func (c *Connection) SetWindows(v bool) {
	c.knowOs = true
	c.isWindows = v
}

// IsWindows is true when SetWindows(true) has been used
func (c *Connection) IsWindows() bool {
	return c.isWindows
}

// Connect opens the SSH connection
func (c *Connection) Connect() error {
	key, err := util.LoadExternalFile(c.KeyPath)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            c.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	address := fmt.Sprintf("%s:%d", c.Address, c.Port)

	sshAgentSock := os.Getenv("SSH_AUTH_SOCK")
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil && sshAgentSock == "" {
		return err
	}
	if err == nil {
		config.Auth = append(config.Auth, ssh.PublicKeys(signer))
	}

	if sshAgentSock != "" {
		sshAgent, err := net.Dial("unix", sshAgentSock)
		if err != nil {
			return fmt.Errorf("cannot connect to SSH agent auth socket %s: %s", sshAgentSock, err)
		}
		log.Debugf("using SSH auth sock %s", sshAgentSock)
		config.Auth = append(config.Auth, ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers))
	}

	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	c.client = client
	go c.keepAlive()
	return nil
}

func (c *Connection) keepAlive() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			log.Tracef("%s: sending ssh keepalive", c.Address)
			_, _, _ = c.client.SendRequest("keepalive@launchpad", true, nil)
		case <-c.done:
			return
		}
	}
}

// Exec executes a command on the host
func (c *Connection) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if len(o.Stdin) == 0 && c.knowOs && !c.isWindows {
		// Only request a PTY when there's no STDIN data, because
		// then you would need to send a CTRL-D after input to signal
		// the end of text
		modes := ssh.TerminalModes{ssh.ECHO: 0}
		err = session.RequestPty("xterm", 80, 40, modes)
		if err != nil {
			return err
		}
	}

	o.LogCmd(c.Address, cmd)

	stdin, _ := session.StdinPipe()
	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()

	multiReader := io.MultiReader(stdout, stderr)
	outputScanner := bufio.NewScanner(multiReader)

	err = session.Start(cmd)

	if len(o.Stdin) > 0 {
		o.LogStdin(c.Address)
		io.WriteString(stdin, o.Stdin)
	}
	stdin.Close()

	for outputScanner.Scan() {
		text := outputScanner.Text()
		stripped := stripansi.Strip(text)
		if stripped != "" {
			o.AddOutput(c.Address, stripped+"\n")
		}
	}

	if err := outputScanner.Err(); err != nil {
		o.LogErrorf("%s: %s", c.Address, err.Error())
	}

	log.Tracef("%s: waiting for command exit", c.Address)
	return session.Wait()
}

// Upload uploads a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (c *Connection) Upload(src, dst string) error {
	if c.IsWindows() {
		return c.uploadWindows(src, dst)
	}
	return c.uploadLinux(src, dst)
}
