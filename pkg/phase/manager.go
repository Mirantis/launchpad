package phase

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Mirantis/launchpad/pkg/analytics"
	"github.com/logrusorgru/aurora/v4"
	event "github.com/segmentio/analytics-go/v3"
	log "github.com/sirupsen/logrus"
)

type phase interface {
	Run() error
	Title() string
}

type withconfig interface {
	Prepare(interface{}) error
}

type withcleanup interface {
	CleanUp()
}

type conditional interface {
	ShouldRun() bool
}

type cleanupdisabling interface {
	DisableCleanup()
}

// Manager executes phases to construct the cluster.
type Manager struct {
	phases       []phase
	config       interface{}
	IgnoreErrors bool
	SkipCleanup  bool
	// Deadline bounds the total wall-clock time Run may spend across all
	// phases. Zero means no deadline. A phase that is still running when the
	// deadline elapses is abandoned (its goroutine is not force-stopped) and
	// Run returns an error naming that phase; the caller is expected to be a
	// short-lived CLI process that exits shortly after, taking the abandoned
	// goroutine down with it.
	Deadline time.Duration
}

// NewManager constructs new phase manager.
func NewManager(config interface{}) *Manager {
	phaseMgr := &Manager{
		config: config,
	}

	return phaseMgr
}

// AddPhases add multiple phases to manager in one call.
func (m *Manager) AddPhases(phases ...phase) {
	m.phases = append(m.phases, phases...)
}

// AddPhase adds a Phase to Manager.
func (m *Manager) AddPhase(p phase) {
	m.phases = append(m.phases, p)
}

// Run executes all the added Phases in order. If Deadline is set, the total
// time spent across all phases is bounded; a phase still running when the
// deadline elapses causes Run to return an error naming that phase instead
// of blocking forever.
func (m *Manager) Run() error {
	var deadlineAt time.Time
	if m.Deadline > 0 {
		deadlineAt = time.Now().Add(m.Deadline)
	}

	for _, phase := range m.phases {
		title := phase.Title()

		if p, ok := phase.(withconfig); ok {
			log.Debugf("preparing phase '%s'", title)
			if err := p.Prepare(m.config); err != nil {
				return fmt.Errorf("phase '%s' failed to prepare: %w", title, err)
			}
		}

		if m.SkipCleanup {
			if p, ok := phase.(cleanupdisabling); ok {
				log.Debugf("disabling in-phase cleanup for '%s'", title)
				p.DisableCleanup()
			}
		}

		if p, ok := phase.(conditional); ok {
			if !p.ShouldRun() {
				log.Debugf("skipping phase '%s'", title)
				continue
			}
		}

		text := aurora.Green("==> Running phase: %s").String()
		log.Infof(text, title)
		start := time.Now()

		timedOut, result := m.runPhase(phase, title, deadlineAt)
		if timedOut {
			return fmt.Errorf("exceeded overall deadline of %s while running phase %q", m.Deadline, title)
		}

		duration := time.Since(start)
		log.Debugf("phase '%s' took %s", title, duration.Truncate(time.Minute))

		if e, ok := phase.(Eventable); ok {
			r := reflect.ValueOf(m.config).Elem()
			props := event.Properties{
				"kind":        r.FieldByName("Kind").String(),
				"api_version": r.FieldByName("APIVersion").String(),
				"duration":    duration.Seconds(),
			}
			for k, v := range e.GetEventProperties() {
				props[k] = v
			}
			props["success"] = result == nil
			defer func() { analytics.TrackEvent(title, props) }()
		}

		if result != nil {
			if p, ok := phase.(withcleanup); ok {
				if !m.SkipCleanup {
					defer p.CleanUp()
				}
			}

			if m.IgnoreErrors {
				log.Debugf("ignoring phase '%s' error: %s", title, result.Error())
				return nil
			}
			return fmt.Errorf("phase failure: %s => %w", title, result)
		}
		log.Debugf("phase '%s' completed successfully", title)
	}

	return nil
}

// runPhase runs a single phase, optionally racing it against deadlineAt. It
// returns the phase's error and whether the deadline elapsed first. On
// timeout the phase's goroutine is left running; the caller (a short-lived
// CLI process) is expected to exit shortly after, which reclaims it. Cleanup
// hooks are not invoked on timeout since the phase never signaled it stopped
// touching shared state.
func (m *Manager) runPhase(target phase, title string, deadlineAt time.Time) (timedOut bool, result error) {
	if deadlineAt.IsZero() {
		if err := target.Run(); err != nil {
			return false, fmt.Errorf("%w", err)
		}
		return false, nil
	}

	remaining := time.Until(deadlineAt)
	if remaining <= 0 {
		return true, nil
	}

	done := make(chan error, 1)
	go func() {
		done <- target.Run()
	}()

	timer := time.NewTimer(remaining)
	defer timer.Stop()

	select {
	case err := <-done:
		if err != nil {
			return false, fmt.Errorf("%w", err)
		}
		return false, nil
	case <-timer.C:
		log.Errorf("phase '%s' exceeded the overall deadline of %s", title, m.Deadline)
		return true, nil
	}
}
