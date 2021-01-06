package configurer

import "github.com/k0sproject/rig/exec"

// Host defines an interface for a host that can be used as a host for a configurer
type Host interface {
	String() string
	Exec(cmd string, opts ...exec.Option) error
	ExecWithOutput(cmd string, opts ...exec.Option) (string, error)
	ExecAll(cmds []string) error
	WriteFileLarge(string, string) error
}
