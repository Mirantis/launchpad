package connection

import (
	"github.com/Mirantis/mcc/pkg/exec"
)

// Connection is an interface to remote host connections
type Connection interface {
	Connect() error
	Disconnect()
	Upload(source string, destination string) error
	IsWindows() bool
	Exec(string, ...exec.Option) error
	ExecInteractive(string) error
	SetName(string)
	String() string
}
