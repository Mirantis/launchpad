package ssh

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	ssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	util "github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

type Configuration struct {
}

type Connection struct {
	Address string
	User    string
	Port    int
	KeyPath string

	isWindows bool
	client    *ssh.Client
}

func (c *Connection) Disconnect() {
	c.client.Close()
}

func (c *Connection) SetWindows(v bool) {
	c.isWindows = v
}

func (c *Connection) IsWindows() bool {
	return c.isWindows
}

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

	return nil
}

func (c *Connection) ExecCmd(cmd string, stdin string, streamStdout bool, sensitiveCommand bool) error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if stdin == "" && !c.isWindows {
		// FIXME not requesting a pty for commands with stdin input for now,
		// as it appears the pipe doesn't get closed with stdinpipe.Close()
		modes := ssh.TerminalModes{}
		err = session.RequestPty("xterm", 80, 40, modes)
		if err != nil {
			return err
		}
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return err
	}

	if !sensitiveCommand {
		log.Debugf("%s: executing command: %s", c.Address, cmd)
	}

	if err := session.Start(cmd); err != nil {
		return err
	}

	if stdin != "" {
		log.Debugf("%s: writing data to command stdin: %s", c.Address, stdin)
		go func() {
			defer stdinPipe.Close()
			io.WriteString(stdinPipe, stdin)
		}()
	}

	multiReader := io.MultiReader(stdout, stderr)
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

	return session.Wait()
}

// ExecWithOutput execs a command on the host and return output
func (c *Connection) ExecWithOutput(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return trimOutput(output), err
	}

	return trimOutput(output), nil
}

func trimOutput(output []byte) string {
	if len(output) == 0 {
		return ""
	}

	return strings.TrimSpace(string(output))
}
