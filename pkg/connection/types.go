package connection

import (
	"github.com/Mirantis/mcc/pkg/exec"
)

// Connection is an interface to remote host connections
type Connection interface {
	Connect() error
	Disconnect()
	SetWindows(bool)
	Upload(source string, destination string) error
	IsWindows() bool
	Exec(string, ...exec.Option) error
	SetName(string)
	String() string
}
