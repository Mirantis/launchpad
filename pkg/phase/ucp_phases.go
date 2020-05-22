package phase

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
)

// Manager executes phases to construct the cluster
type Manager struct {
	phases []Phase
	config *config.ClusterConfig
}

// NewManager constructs new phase manager
func NewManager(config *config.ClusterConfig) *Manager {
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
		text := aurora.Green("==> Running phase: %s").String()
		log.Infof(text, phase.Title())
		start := time.Now()
		props := analytics.NewAnalyticsEventProperties()
		duration := time.Since(start)
		err := phase.Run(m.config)
		props["duration"] = duration.Seconds()
		for k, v := range phase.GetEventProperties() {
			props[k] = v
		}
		if err != nil {
			props["success"] = false
			analytics.TrackEvent(phase.GetEventTitle(), props)
			return err
		}
		props["success"] = true
		analytics.TrackEvent(phase.GetEventTitle(), props)

	}

	return nil
}
