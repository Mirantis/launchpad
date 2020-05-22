package install

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	log "github.com/sirupsen/logrus"
)

// Install ...
func Install(ctx *cli.Context) error {
	if err := analytics.RequireRegisteredUser(); err != nil {
		return err
	}
	configFile := ctx.String("config")
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
		os.Stdout.WriteString("   Mirantis Launchpad\n\n")
	}

	log.Debugf("loaded cluster cfg: %+v", clusterConfig)

	phaseManager := phase.NewManager(&clusterConfig)
	phaseManager.AddPhase(&phase.InitState{})
	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{})
	phaseManager.AddPhase(&phase.PrepareHost{})
	phaseManager.AddPhase(&phase.InstallEngine{})
	phaseManager.AddPhase(&phase.PullImages{})
	phaseManager.AddPhase(&phase.InitSwarm{})
	phaseManager.AddPhase(&phase.InstallUCP{})
	phaseManager.AddPhase(&phase.UpgradeUcp{})
	phaseManager.AddPhase(&phase.JoinManagers{})
	phaseManager.AddPhase(&phase.JoinWorkers{})
	phaseManager.AddPhase(&phase.SaveState{})
	phaseManager.AddPhase(&phase.Disconnect{})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}
	props := analytics.NewAnalyticsEventProperties()
	props["kind"] = clusterConfig.Kind
	props["api_version"] = clusterConfig.APIVersion
	props["hosts"] = len(clusterConfig.Spec.Hosts)
	props["managers"] = len(clusterConfig.Spec.Managers())
	props["workers"] = len(clusterConfig.Spec.Workers())
	props["engine_version"] = clusterConfig.Spec.Engine.Version
	clusterID := clusterConfig.State.ClusterID
	props["cluster_id"] = clusterID
	// send ucp analytics user id as ucp_instance_id property
	ucpInstanceID := fmt.Sprintf("%x", sha1.Sum([]byte(clusterID)))
	props["ucp_instance_id"] = ucpInstanceID
	analytics.TrackEvent("Cluster Installed", props)
	return nil
}
