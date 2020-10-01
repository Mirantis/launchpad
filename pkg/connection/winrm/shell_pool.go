package winrm

import (
	"runtime"
	"sync"

	"github.com/jbrekelmans/winrm"
	"github.com/lithammer/shortuuid"
	log "github.com/sirupsen/logrus"
)

// ShellLease is a lease from the winrm shell pool. use .Release() when done
type ShellLease struct {
	id    string
	shell *winrm.Shell
	pool  *ShellPool
}

// Release puts the shell back into the pool
func (l *ShellLease) Release() {
	l.pool.Put(l)
}

// ShellPool is a sync.Pool based simple pool for obtaining winrm shells. There's no limit to the number of shells.
type ShellPool struct {
	client *winrm.Client
	pool   *sync.Pool
}

// Get returns a new or idle ShellLease instance
func (s *ShellPool) Get() *ShellLease {
	item := s.pool.Get().(*ShellLease)
	log.Tracef("received lease %s from pool", item.id)
	return item
}

// Put puts a ShellLease back into the pool
func (s *ShellPool) Put(item *ShellLease) {
	log.Tracef("returning lease %s back to pool", item.id)
	s.pool.Put(item)
}

/// should close the shell when garbagecollecting
func finalizer(item *ShellLease) {
	item.shell.Close()
}

// creates new leases
func (s *ShellPool) factory() interface{} {
	shell, err := s.client.CreateShell()
	if err != nil {
		return nil
	}

	lease := &ShellLease{
		id:    shortuuid.New(),
		shell: shell,
		pool:  s,
	}
	runtime.SetFinalizer(lease, func(l *ShellLease) { l.shell.Close() })
	log.Tracef("created a new lease %s", lease.id)
	return lease
}

// NewShellPool returns a new shellpool for a winrm client
func NewShellPool(client *winrm.Client) *ShellPool {
	pool := &ShellPool{
		client: client,
	}

	pool.pool = &sync.Pool{New: pool.factory}
	return pool
}
