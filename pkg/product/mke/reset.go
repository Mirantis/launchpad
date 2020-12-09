package mke

import (
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	mke "github.com/Mirantis/mcc/pkg/product/mke/phase"
)

// Reset uninstalls a Docker Enterprise cluster
func (p *MKE) Reset() error {
	phaseManager := phase.NewManager(&p.ClusterConfig)

	phaseManager.AddPhases(
		&common.Connect{},
		&mke.GatherFacts{},
		&common.RunHooks{Stage: "Before", Action: "Reset", StepListFunc: func(h *api.Host) *[]string {
			if h.Hooks == nil || h.Hooks.Reset == nil || h.Hooks.Reset.Before == nil {
				return &[]string{}
			}
			return h.Hooks.Reset.Before
		}},

		// begin MSR phases
		&mke.UninstallMSR{},
		// end MSR phases

		&mke.UninstallMKE{},
		&mke.DownloadInstaller{},
		&mke.UninstallEngine{},
		&mke.CleanUp{},
		&common.RunHooks{Stage: "After", Action: "Reset", StepListFunc: func(h *api.Host) *[]string {
			if h.Hooks == nil || h.Hooks.Reset == nil || h.Hooks.Reset.After == nil {
				return &[]string{}
			}

			return h.Hooks.Reset.After
		}},
		&common.Disconnect{},
	)

	return phaseManager.Run()
}
