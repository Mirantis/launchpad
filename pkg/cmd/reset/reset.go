package reset

import (
	"fmt"
	"os"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/mattn/go-isatty"

	log "github.com/sirupsen/logrus"
)

// Reset ...
func Reset(configFile string) error {
	cfgData, err := config.ResolveClusterFile(configFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = config.Validate(&clusterConfig); err != nil {
		return err
	}

	if isatty.IsTerminal(os.Stdout.Fd()) {
		os.Stdout.WriteString(util.Logo)
		os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
	}

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	phaseManager := phase.NewManager(&clusterConfig)

	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{})
	phaseManager.AddPhase(&phase.RunHooks{Stage: "Before", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.Before }})
	phaseManager.AddPhase(&phase.UninstallDTR{})
	phaseManager.AddPhase(&phase.UninstallUCP{})
	phaseManager.AddPhase(&phase.DownloadInstaller{})
	phaseManager.AddPhase(&phase.UninstallEngine{})
	phaseManager.AddPhase(&phase.CleanUp{})
	phaseManager.AddPhase(&phase.RunHooks{Stage: "After", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.After }})
	phaseManager.AddPhase(&phase.Disconnect{})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}

	return nil
}
