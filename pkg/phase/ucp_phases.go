package phase

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/sirupsen/logrus"
)

type PhaseManager struct {
	phases []Phase
	config *config.ClusterConfig
}

func NewPhaseManager(config *config.ClusterConfig) *PhaseManager {
	phaseMgr := &PhaseManager{
		config: config,
	}

	return phaseMgr
}

// AddPhase adds a Phase to PhaseManager
func (m *PhaseManager) AddPhase(phase Phase) {
	m.phases = append(m.phases, phase)
}

// Run executes all the added Phases in order
func (m *PhaseManager) Run() error {
	for _, phase := range m.phases {
		logrus.Infof("==> Running phase: %s", phase.Title())
		err := phase.Run(m.config)
		if err != nil {
			return err
		}
	}

	return nil
}
