package winrm

import (
	"runtime"
	"sync"

	"github.com/jbrekelmans/winrm"
	"github.com/lithammer/shortuuid"
	log "github.com/sirupsen/logrus"
)

type ShellLease struct {
	id    string
	shell *winrm.Shell
	pool  *ShellPool
}

func (l *ShellLease) Release() {
	l.pool.Put(l)
}

type ShellPool struct {
	client *winrm.Client
	pool   *sync.Pool
}

func (s *ShellPool) Get() *ShellLease {
	item := s.pool.Get().(*ShellLease)
	log.Tracef("received lease %s from pool", item.id)
	return item
}

func (s *ShellPool) Put(item *ShellLease) {
	log.Tracef("returning lease %s back to pool", item.id)
	s.pool.Put(item)
}

func finalizer(item *ShellLease) {
	item.shell.Close()
}

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

func NewShellPool(client *winrm.Client) *ShellPool {
	pool := &ShellPool{
		client: client,
	}

	pool.pool = &sync.Pool{New: pool.factory}
	return pool
}
