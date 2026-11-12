package phase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	retry "github.com/avast/retry-go"
	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

type connectable interface {
	Connect(context.Context) error
	String() string
	Exec(cmd string, opts ...cmd.ExecOption) error
}

// Connect connects to each of the hosts.
type Connect struct {
	hosts []connectable
}

// Prepare digs out the hosts from the config.
func (p *Connect) Prepare(config interface{}) error {
	r := reflect.ValueOf(config).Elem()
	spec := r.FieldByName("Spec").Elem()
	hosts := spec.FieldByName("Hosts")
	for i := 0; i < hosts.Len(); i++ {
		if h, ok := hosts.Index(i).Interface().(connectable); ok {
			p.hosts = append(p.hosts, h)
		}
	}

	return nil
}

// ShouldRun is true when there are hosts that need to be connected.
func (p *Connect) ShouldRun() bool {
	return len(p.hosts) > 0
}

// Title for the phase.
func (p *Connect) Title() string {
	return "Open Remote Connection"
}

// Run connects to all the hosts in parallel.
func (p *Connect) Run() error {
	var (
		wg     sync.WaitGroup
		result error
		mu     sync.Mutex
	)

	for _, h := range p.hosts {
		wg.Add(1)
		go func(h connectable) {
			defer wg.Done()
			if err := p.connectHost(h); err != nil {
				mu.Lock()
				result = errors.Join(result, fmt.Errorf("connect %s: %w", h, err))
				mu.Unlock()
			}
		}(h)
	}

	wg.Wait()

	if result != nil {
		return fmt.Errorf("failed to connect all hosts: %w", result)
	}

	return nil
}

const retries = 60

func (p *Connect) connectHost(host connectable) error {
	err := retry.Do(
		func() error {
			if err := host.Connect(context.Background()); err != nil {
				return fmt.Errorf("connect: %w", err)
			}
			return nil
		},
		retry.OnRetry(
			func(n uint, err error) {
				log.Errorf("%s: attempt %d of %d.. failed to connect: %s", host, n+1, retries, err.Error())
			},
		),
		retry.RetryIf(shouldRetryConnect),
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(retries),
	)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return p.testConnection(host)
}

// shouldRetryConnect reports whether a failed connect attempt is worth another
// try. Everything is retried except states no amount of waiting will change.
//
// A credential rejection is retried, and rig decides that rather than this
// phase. Protocol implementations tag it with protocol.ErrAuthFailed -- an
// HTTP 401/403 on WinRM, an exhausted handshake on SSH -- and rig deliberately
// does not wrap that in ErrNonRetryable, because a rejection is routinely
// transient while a host is still being provisioned: the WinRM listener
// answers before authentication is configured, and sshd accepts connections
// before authorized_keys is written.
//
// Launchpad used to match "http error 401" and friends by substring here,
// because rig v2.1 classified 401/403 as ErrNonRetryable and aborted this wait
// on its first attempt (smoke-fips regressed on exactly that). rig v2.2
// carries that fix, so the substring matching is gone. See PRODENG-3594 and
// k0sproject/rig#421.
func shouldRetryConnect(err error) bool {
	return !errors.Is(err, rig.ErrNonRetryable)
}

func (p *Connect) testConnection(h connectable) error {
	log.Infof("%s: testing connection", h)

	if err := h.Exec("echo"); err != nil {
		return fmt.Errorf("failed to test connection to %s: %w", h, err)
	}

	return nil
}
