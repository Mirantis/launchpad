package phase

import (
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
)

// DownloadBundle phase downloads the client bundle to local storage.
type DownloadBundle struct {
	phase.BasicPhase
}

// Title for the phase.
func (p *DownloadBundle) Title() string {
	return "Download Client Bundle"
}

// Run collect all the facts from hosts in parallel.
func (p *DownloadBundle) Run() error {
	return mke.DownloadBundle(p.Config)
}
