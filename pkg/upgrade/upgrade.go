package upgrade

import (
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Upgrade upgrades an existing cluster
func Upgrade(ctx *cli.Context) error {
	cfgData, err := util.ResolveClusterFile(ctx)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	if err = clusterConfig.Validate(); err != nil {
		return err
	}

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	phaseManager := phase.NewManager(&clusterConfig)

	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherHostFacts{})
	phaseManager.AddPhase(&phase.PrepareHost{})
	// Currently install engine handles the upgrade path too
	phaseManager.AddPhase(&phase.InstallEngine{})
	phaseManager.AddPhase(&phase.GatherUcpFacts{})
	phaseManager.AddPhase(&phase.PullImages{})
	phaseManager.AddPhase(&phase.UpgradeUcp{})
	phaseManager.AddPhase(&phase.JoinManagers{})
	phaseManager.AddPhase(&phase.JoinWorkers{})
	phaseManager.AddPhase(&phase.Disconnect{})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}

	return nil

}
