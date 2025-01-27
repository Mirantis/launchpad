package phase

import (
	"fmt"

	github.com/Mirantis/launchpad/pkg/mke"
	github.com/Mirantis/launchpad/pkg/phase"
)

// DownloadBundle phase downloads the client bundle to local storage.
type DownloadBundle struct {
	phase.BasicPhase
}

// Title for the phase.
func (p *DownloadBundle) Title() string {
	return "Download Client Bundle"
}

var errInvalidConfig = fmt.Errorf("invalid config")

// Run collect all the facts from hosts in parallel.
func (p *DownloadBundle) Run() error {
	if err := mke.DownloadBundle(p.Config); err != nil {
		return fmt.Errorf("failed to download client bundle: %w", err)
	}

	return nil
}
