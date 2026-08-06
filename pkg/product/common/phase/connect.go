package phase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
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
func shouldRetryConnect(err error) bool {
	if isTransientAuthError(err) {
		return true
	}

	return !errors.Is(err, rig.ErrNonRetryable)
}

// isTransientAuthError reports whether err is a WinRM HTTP 401/403.
//
// This phase exists to wait for hosts to become reachable, and on Windows an
// auth rejection is an expected part of that wait: the WinRM HTTPS listener
// starts answering before provisioning has finished configuring authentication,
// so a freshly booted host returns 401 for a while with entirely correct
// credentials.
//
// rig v2 wraps 401/403 as protocol.ErrNonRetryable, which is reasonable for a
// general purpose library but wrong here -- it aborts the wait on its first
// attempt. rig v0 did not classify these, so launchpad retried them and the
// condition healed itself; smoke-fips regressed on exactly this. Treat them as
// retryable again, while still honouring ErrNonRetryable for everything else
// (bad certificates, host key mismatches, misconfigured bastions), which no
// amount of waiting will fix.
//
// The cost is that genuinely wrong credentials now take the full retry budget
// to report instead of failing immediately. That is the right trade for a
// provisioning tool: a cluster that would have come up must not fail, and the
// budget is bounded at a few minutes.
//
// Matching is by substring because the underlying WinRM library returns untyped
// formatted errors and rig exports no sentinel for them. rig's own isAuthError
// does the same thing, and both message shapes it covers are matched here. See
// PRODENG-3594.
func isTransientAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, probe := range []string{
		"http error 401",
		"http error 403",
		"http response error: 401",
		"http response error: 403",
	} {
		if strings.Contains(msg, probe) {
			return true
		}
	}

	return false
}

func (p *Connect) testConnection(h connectable) error {
	log.Infof("%s: testing connection", h)

	if err := h.Exec("echo"); err != nil {
		return fmt.Errorf("failed to test connection to %s: %w", h, err)
	}

	return nil
}
