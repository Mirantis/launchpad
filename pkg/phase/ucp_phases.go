package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/logrusorgru/aurora"
	log "github.com/sirupsen/logrus"
)

// Manager executes phases to construct the cluster
type Manager struct {
	phases []Phase
	config *api.ClusterConfig
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
		text := aurora.Green("==> Running phase: %s").String()
		log.Infof(text, phase.Title())
		err := phase.Run(m.config)
		if err != nil {
			return err
		}
	}

	return nil
}
