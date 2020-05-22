package phase

import (
	"fmt"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta1"
	"github.com/Mirantis/mcc/pkg/state"
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
func (p *SaveState) Run(config *api.ClusterConfig) error {
	p.EventTitle = "Local State Saved"
	if config.State == nil {
		return fmt.Errorf("internal state was nil, this should not happen")
	}

	config.State.ClusterID = config.Spec.Ucp.Metadata.ClusterID

	return state.Save(config.State)
}
