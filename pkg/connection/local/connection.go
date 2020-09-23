package local

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

const hostname = "localhost"

// Connection is a direct localhost connection
type Connection struct{}

// NewConnection returns a new connection
func NewConnection() *Connection {
	return &Connection{}
}

// SetWindows on local connection does nothing
func (c *Connection) SetWindows(bool) {}

// IsWindows is true when SetWindows(true) has been used
func (c *Connection) IsWindows() bool {
	return runtime.GOOS == "windows"
}

// Connect on local connection does nothing
func (c *Connection) Connect() error {
	return nil
}

// Disconnect on local connection does nothing
func (c *Connection) Disconnect() {}

// ExecCmd executes a command on the host
func (c *Connection) ExecCmd(cmd string, stdin string, streamStdout bool, sensitiveCommand bool) error {
	command := c.command(cmd)

	if stdin != "" {
		if sensitiveCommand || len(stdin) > 256 {
			log.Debugf("%s: writing %d bytes to command stdin", hostname, len(stdin))
		} else {
			log.Debugf("%s: writing %d bytes to command stdin: %s", hostname, len(stdin), stdin)
		}

		command.Stdin = strings.NewReader(stdin)
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	multiReader := io.MultiReader(stdout, stderr)
	outputScanner := bufio.NewScanner(multiReader)

	if !sensitiveCommand {
		log.Debugf("%s: executing command: %s", hostname, cmd)
	}

	command.Start()

	for outputScanner.Scan() {
		if streamStdout {
			log.Infof("%s: %s", hostname, outputScanner.Text())
		} else {
			log.Debugf("%s: %s", hostname, outputScanner.Text())
		}
	}

	return command.Wait()
}

// ExecWithOutput execs a command on the host and returns its output
func (c *Connection) ExecWithOutput(cmd string) (string, error) {
	command := c.command(cmd)
	output, err := command.CombinedOutput()
	return trimOutput(output), err
}

func trimOutput(output []byte) string {
	if len(output) == 0 {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func (c *Connection) command(cmd string) *exec.Cmd {
	if c.IsWindows() {
		return exec.Command(cmd)
	}

	return exec.Command("bash", "-c", "--", cmd)
}

// ExecInteractive executes a command on the host and copies stdin/stdout/stderr from local host
func (c *Connection) ExecInteractive() error {
	command := c.command("bash -s")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	command.Start()

	return command.Wait()
}
