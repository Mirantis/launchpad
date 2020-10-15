package phase

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Manager executes phases to construct the cluster
type Manager struct {
	phases       []Phase
	config       *api.ClusterConfig
	IgnoreErrors bool
	SkipCleanup  bool
}

// NewManager constructs new phase manager
func NewManager(config *api.ClusterConfig) *Manager {
	phaseMgr := &Manager{
		config: config,
	}

	return phaseMgr
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
			props := event.Properties{
				"kind":        m.config.Kind,
				"api_version": m.config.APIVersion,
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
