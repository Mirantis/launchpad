package phase

import (
	"os"

	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/state"
	log "github.com/sirupsen/logrus"
)

// InitState loads or initializes the state
type InitState struct {
	Analytics
}

// Title title for the phase
func (p *InitState) Title() string {
	return "Load or init local state"
}

// Run runs the state management logic
func (p *InitState) Run(config *config.ClusterConfig) error {
	localState, err := state.LoadState(config.Name)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("Local state not found, initializing")
			localState, err = state.InitState(config.Name)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	config.State = localState
	log.Debugf("Initialized local state: %+v", config.State)

	return nil
}
