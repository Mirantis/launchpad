package phase

import (
	"reflect"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Phase interface
type Phase interface {
	Eventable
	Run() error
	Title() string
	Prepare(interface{}) error
	ShouldRun() bool
	CleanUp()
}

// Manager executes phases to construct the cluster
type Manager struct {
	phases       []Phase
	config       interface{}
	IgnoreErrors bool
	SkipCleanup  bool
}

// NewManager constructs new phase manager
func NewManager(config interface{}) *Manager {
	phaseMgr := &Manager{
		config: config,
	}

	return phaseMgr
}

// AddPhases add multiple phases to manager in one call
func (m *Manager) AddPhases(phases ...Phase) {
	m.phases = append(m.phases, phases...)
}

// AddPhase adds a Phase to Manager
func (m *Manager) AddPhase(phase Phase) {
	m.phases = append(m.phases, phase)
}

// Run executes all the added Phases in order
func (m *Manager) Run() error {
	for _, phase := range m.phases {
		log.Debugf("preparing phase '%s'", phase.Title())
		err := phase.Prepare(m.config)
		if err != nil {
			return err
		}

		if !phase.ShouldRun() {
			log.Debugf("skipping phase '%s'", phase.Title())
			continue
		}

		text := aurora.Green("==> Running phase: %s").String()
		log.Infof(text, phase.Title())
		if p, ok := interface{}(phase).(Eventable); ok {
			start := time.Now()
			r := reflect.ValueOf(m.config).Elem()
			props := event.Properties{
				"kind":        r.FieldByName("Kind").String(),
				"api_version": r.FieldByName("APIVersion").String(),
			}

			err := phase.Run()

			duration := time.Since(start)
			props["duration"] = duration.Seconds()
			for k, v := range p.GetEventProperties() {
				props[k] = v
			}
			if err != nil {
				props["success"] = false
				analytics.TrackEvent(phase.Title(), props)
				if !m.IgnoreErrors {
					return err
				}
			}
			props["success"] = true
			analytics.TrackEvent(phase.Title(), props)

		} else {
			err := phase.Run()
			if err != nil && !m.IgnoreErrors {
				return err
			}
			if !m.SkipCleanup {
				defer phase.CleanUp()
			}
		}
	}

	return nil
}
