package phase

import (
	"errors"
	"fmt"

	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
)

var errInvalidConfig = errors.New("invalid config")

// DownloadBundle phase downloads the client bundle to local storage if
// the bundle is not already present.
type DownloadBundle struct {
	phase.BasicPhase
}

// Title for the phase.
func (p *DownloadBundle) Title() string {
	return "Download Client Bundle"
}

// Run collect all the facts from hosts in parallel.
func (p *DownloadBundle) Run() error {
	if err := mke.DownloadBundle(p.Config); err != nil {
		return fmt.Errorf("failed to download client bundle: %w", err)
	}

	return nil
}
