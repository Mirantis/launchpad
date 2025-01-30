package mke

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/phase"
	common "github.com/Mirantis/launchpad/pkg/product/common/phase"
	mke "github.com/Mirantis/launchpad/pkg/product/mke/phase"
)

// Reset uninstalls a Docker Enterprise cluster.
func (p *MKE) Reset() error {
	phaseManager := phase.NewManager(&p.ClusterConfig)

	phaseManager.AddPhases(
		&mke.OverrideHostSudo{},
		&common.Connect{},
		&mke.DetectOS{},
		&mke.GatherFacts{},
		&mke.PrepareHost{},
		&common.RunHooks{Stage: "before", Action: "reset"},

		// begin MSR phases
		&mke.UninstallMSR{},
		// end MSR phases

		&mke.UninstallMKE{},
		&mke.DownloadInstaller{},
		&mke.UninstallMCR{},
		&mke.CleanUp{},
		&common.RunHooks{Stage: "after", Action: "reset"},
		&common.Disconnect{},
	)

	if err := phaseManager.Run(); err != nil {
		return fmt.Errorf("reset failed: %w", err)
	}
	return nil
}
