package dockerenterprise

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
)

// Reset uninstalls a Docker Enterprise cluster
func (p *DockerEnterprise) Reset() error {
	phaseManager := phase.NewManager(&p.ClusterConfig)

	phaseManager.AddPhases(
		&phase.Connect{},
		&phase.GatherFacts{},
		&phase.RunHooks{Stage: "Before", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.Before }},
		// begin DTR phases
		&phase.UninstallDTR{},
		// end DTR phases
		&phase.UninstallUCP{},
		&phase.DownloadInstaller{},
		&phase.UninstallEngine{},
		&phase.CleanUp{},
		&phase.RunHooks{Stage: "After", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.After }},
		&phase.Disconnect{},
	)

	return phaseManager.Run()
}
