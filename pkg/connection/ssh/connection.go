package ssh

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"sync"
	"time"

	ssh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/Mirantis/mcc/pkg/exec"
	util "github.com/Mirantis/mcc/pkg/util"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
)

// Connection describes an SSH connection
type Connection struct {
	Address string
	User    string
	Port    int
	KeyPath string

	isWindows bool
	client    *ssh.Client
}

// Disconnect closes the SSH connection
func (c *Connection) Disconnect() {
	c.client.Close()
}

// SetWindows can be used to tell the SSH connection to consider the host to be running Windows
func (c *Connection) SetWindows(v bool) {
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

	return nil
}

// Exec executes a command on the host
func (c *Connection) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if o.Stdin == "" && !c.isWindows {
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

	o.LogCmd(c.Address, cmd)

	if err := session.Start(cmd); err != nil {
		return err
	}

	if o.Stdin != "" {
		o.LogStdin(c.Address)

		go func() {
			defer stdinPipe.Close()
			io.WriteString(stdinPipe, o.Stdin)
		}()
	}

	multiReader := io.MultiReader(stdout, stderr)
	outputScanner := bufio.NewScanner(multiReader)

	for outputScanner.Scan() {
		o.AddOutput(c.Address, outputScanner.Text()+"\n")
	}

	if err := outputScanner.Err(); err != nil {
		o.LogErrorf("%s:  %s", c.Address, err.Error())
	}

	return session.Wait()
}

// WriteFileLarge copies a larger file to the host.
// Use instead of configurer.WriteFile when it seems appropriate
func (c *Connection) WriteFileLarge(src, dstdir string) error {
	startTime := time.Now()

	stat, err := os.Stat(src)
	if err != nil {
		return err
	}
	base := path.Base(src)

	log.Infof("%s: uploading %s to %s/%s", c.Address, util.FormatBytes(uint64(stat.Size())), dstdir, base)

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	wg := sync.WaitGroup{}
	wg.Add(1)

	hostIn, err := session.StdinPipe()
	if err != nil {
		return err
	}

	gw, err := gzip.NewWriterLevel(hostIn, gzip.BestSpeed)
	if err != nil {
		return err
	}

	go func() {
		defer wg.Done()
		defer gw.Close()
		io.Copy(gw, in)
	}()

	err = session.Start(fmt.Sprintf(`gzip -d > %s`, shellescape.Quote(dstdir+"/"+base)))
	if err != nil {
		return err
	}
	wg.Wait()
	hostIn.Close()
	session.Wait()
	duration := time.Since(startTime).Seconds()
	speed := float64(stat.Size()) / duration
	log.Debugf("%s: transfered %d bytes in %f seconds (%s/s)", c.Address, stat.Size(), duration, util.FormatBytes(uint64(speed)))
	return err
}
