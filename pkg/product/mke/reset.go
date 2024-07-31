package mke

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	mke "github.com/Mirantis/mcc/pkg/product/mke/phase"
)

// Reset uninstalls a Docker Enterprise cluster.
func (p *MKE) Reset() error {
	phaseManager := phase.NewManager(&p.ClusterConfig)

	phaseManager.AddPhases(
		&common.Connect{},
		&mke.DetectOS{},
		&mke.GatherFacts{},
		&mke.PrepareHost{},
		&common.RunHooks{Stage: "before", Action: "reset"},
	)

	// begin MSR phases
	if p.ClusterConfig.Spec.ContainsMSR2() {
		phaseManager.AddPhase(&mke.UninstallMSR{})
	} else if p.ClusterConfig.Spec.ContainsMSR3() {
		phaseManager.AddPhase(&mke.UninstallMSR3{})
	}

	if p.ClusterConfig.Spec.MKE.Metadata.Installed {
		phaseManager.AddPhases(
			&mke.UninstallMKE{},
		)
	}

	phaseManager.AddPhases(
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
