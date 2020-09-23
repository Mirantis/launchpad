package connection

// Connection is an interface to remote host connections
type Connection interface {
	Connect() error
	Disconnect()
	ExecCmd(cmd string, stdin string, streamStdout bool, sensitiveCommand bool) error
	ExecWithOutput(cmd string) (string, error)
	ExecInteractive() error
	IsWindows() bool
	SetWindows(bool)
}
