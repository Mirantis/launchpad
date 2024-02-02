package phase

import (
	"fmt"
	"reflect"
	"time"

	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"

	"github.com/Mirantis/mcc/pkg/analytics"
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

// Run executes all the added Phases in order.
func (m *Manager) Run() error {
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

		result := phase.Run()

		duration := time.Since(start)
		log.Debugf("phase '%s' took %s", title, duration.Truncate(time.Second))

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
