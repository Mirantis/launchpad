package phase

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

type host interface {
	ExecAll([]string) error
	String() string
}

// RunHooks phase runs a set of hooks configured for the host.
type RunHooks struct {
	Action string
	Stage  string

	steps map[host][]string
}

// Prepare digs out the hosts with steps from the config.
func (p *RunHooks) Prepare(config interface{}) error {
	p.steps = make(map[host][]string)
	r := reflect.ValueOf(config).Elem()
	spec := r.FieldByName("Spec").Elem()
	hosts := spec.FieldByName("Hosts")
	for i := 0; i < hosts.Len(); i++ {
		hostVal := hosts.Index(i)
		hooksF := hostVal.Elem().FieldByName("Hooks")
		if hooksF.IsNil() {
			continue
		}
		hooksI, ok := hooksF.Interface().(common.Hooks)
		if !ok {
			continue
		}
		if action := hooksI[p.Action]; action != nil {
			if steps := action[p.Stage]; steps != nil {
				he, ok := hostVal.Interface().(host)
				if ok {
					p.steps[he] = steps
				}
			}
		}
	}

	return nil
}

// ShouldRun is true when there are hosts that need to be connected.
func (p *RunHooks) ShouldRun() bool {
	return len(p.steps) > 0
}

func ucFirst(s string) string {
	if s == "" {
		return ""
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}

// Title for the phase.
func (p *RunHooks) Title() string {
	return fmt.Sprintf("Run %s %s Hooks", ucFirst(p.Stage), ucFirst(p.Action))
}

// Run does all the prep work on the hosts in parallel.
func (p *RunHooks) Run() error {
	var (
		wg     sync.WaitGroup
		result error
		mu     sync.Mutex
	)

	for h, steps := range p.steps {
		wg.Add(1)
		go func(h host, steps []string) {
			defer wg.Done()
			if err := h.ExecAll(steps); err != nil {
				mu.Lock()
				result = errors.Join(result, fmt.Errorf("%s: %w", h.String(), err))
				mu.Unlock()
			}
		}(h, steps)
	}

	wg.Wait()

	if result != nil {
		return fmt.Errorf("hook execution failed: %w", result)
	}

	return nil
}
