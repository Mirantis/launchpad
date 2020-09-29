package connection

import (
	"github.com/Mirantis/mcc/pkg/exec"
)

// Connection is an interface to remote host connections
type Connection interface {
	Connect() error
	Disconnect()
	SetWindows(bool)
	IsWindows() bool
	Exec(string, ...exec.Option) error
}
