package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/config"
)

// SaveState saves the local state after succesfull run
type SaveState struct {
	Analytics
}

// Title for the phase
func (p *SaveState) Title() string {
	return "Save local state"
}

// Run does the actual saving of the local state file
func (p *SaveState) Run(config *config.ClusterConfig) error {
	if config.State == nil {
		return fmt.Errorf("internal state was nil, this should not happen")
	}

	return config.State.Save()
}
