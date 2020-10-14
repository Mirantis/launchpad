package local

import (
	"bufio"
	"io"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
)

const hostname = "localhost"

// Connection is a direct localhost connection
type Connection struct {
	name string
}

// NewConnection returns a new connection
func NewConnection() *Connection {
	return &Connection{}
}

// SetName sets the connection's printable name
func (c *Connection) SetName(n string) {
	c.name = n
}

// String returns the connection's printable name
func (c *Connection) String() string {
	if c.name == "" {
		return hostname
	}

	return c.name
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

// Exec executes a command on the host
func (c *Connection) Exec(cmd string, opts ...exec.Option) error {
	o := exec.Build(opts...)
	command := c.command(cmd)

	if o.Stdin != "" {
		o.LogStdin(hostname)

		command.Stdin = strings.NewReader(o.Stdin)
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

	o.LogCmd(hostname, cmd)

	command.Start()

	for outputScanner.Scan() {
		o.AddOutput(hostname, outputScanner.Text()+"\n")
	}

	return command.Wait()
}

func (c *Connection) command(cmd string) *osexec.Cmd {
	if c.IsWindows() {
		return osexec.Command(cmd)
	}

	return osexec.Command("bash", "-c", "--", cmd)
}

// Upload copies a larger file to another path on the host.
func (c *Connection) Upload(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	defer out.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return err
}
