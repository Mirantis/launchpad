package dockerenterprise

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	de "github.com/Mirantis/mcc/pkg/product/dockerenterprise/phase"
)

// Reset uninstalls a Docker Enterprise cluster
func (p *DockerEnterprise) Reset() error {
	phaseManager := phase.NewManager(&p.ClusterConfig)

	phaseManager.AddPhases(
		&common.Connect{},
		&de.GatherFacts{},
		&common.RunHooks{Stage: "Before", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.Before }},

		// begin DTR phases
		&de.UninstallDTR{},
		// end DTR phases

		&de.UninstallUCP{},
		&de.DownloadInstaller{},
		&de.UninstallEngine{},
		&de.CleanUp{},
		&common.RunHooks{Stage: "After", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.After }},
		&common.Disconnect{},
	)

	return phaseManager.Run()
}
