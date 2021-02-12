package phase

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
)

type host interface {
	ExecAll([]string) error
	String() string
}

// RunHooks phase runs a set of hooks configured for the host
type RunHooks struct {
	Action string
	Stage  string

	steps map[host][]string
}

// Prepare digs out the hosts with steps from the config
func (p *RunHooks) Prepare(config interface{}) error {
	p.steps = make(map[host][]string)
	r := reflect.ValueOf(config).Elem()
	spec := r.FieldByName("Spec").Elem()
	hosts := spec.FieldByName("Hosts")
	for i := 0; i < hosts.Len(); i++ {
		h := hosts.Index(i)
		hooksF := h.Elem().FieldByName("Hooks")
		if hooksF.IsNil() {
			continue
		}
		hooksI := hooksF.Interface().(common.Hooks)
		if action := hooksI[p.Action]; action != nil {
			if steps := action[p.Stage]; steps != nil {
				he := h.Interface().(host)
				p.steps[he] = steps
			}
		}
	}

	return nil
}

// ShouldRun is true when there are hosts that need to be connected
func (p *RunHooks) ShouldRun() bool {
	return len(p.steps) > 0
}

// Title for the phase
func (p *RunHooks) Title() string {
	return fmt.Sprintf("Run %s %s Hooks", strings.Title(p.Stage), strings.Title(p.Action))
}

// Run does all the prep work on the hosts in parallel
func (p *RunHooks) Run() error {
	var wg sync.WaitGroup
	var errors []string
	type erritem struct {
		host string
		err  error
	}
	ec := make(chan erritem, 1)

	wg.Add(len(p.steps))

	for h, steps := range p.steps {
		go func(h host, steps []string) {
			ec <- erritem{h.String(), h.ExecAll(steps)}
		}(h, steps)
	}

	go func() {
		for e := range ec {
			if e.err != nil {
				errors = append(errors, fmt.Sprintf("%s: %s", e.host, e.err.Error()))
			}
			wg.Done()
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("failed on %d hosts:\n - %s", len(errors), strings.Join(errors, "\n - "))
	}

	return nil
}
