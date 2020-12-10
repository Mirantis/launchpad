package mke

import (
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	mke "github.com/Mirantis/mcc/pkg/product/mke/phase"
)

// Reset uninstalls a Docker Enterprise cluster
func (p *MKE) Reset() error {
	phaseManager := phase.NewManager(&p.ClusterConfig)

	phaseManager.AddPhases(
		&common.Connect{},
		&mke.GatherFacts{},
		&common.RunHooks{Stage: "before", Action: "reset"},

		// begin MSR phases
		&mke.UninstallMSR{},
		// end MSR phases

		&mke.UninstallMKE{},
		&mke.DownloadInstaller{},
		&mke.UninstallEngine{},
		&mke.CleanUp{},
		&common.RunHooks{Stage: "after", Action: "reset"},
		&common.Disconnect{},
	)

	return phaseManager.Run()
}
