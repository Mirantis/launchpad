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

	phaseManager.AddPhase(&phase.InitState{phase.Analytics{"Local State Loaded", nil}})
	phaseManager.AddPhase(&phase.Connect{phase.Analytics{"SSH Connection Opened", nil}})
	phaseManager.AddPhase(&phase.GatherFacts{phase.Analytics{"Facts Gathered", nil}})
	phaseManager.AddPhase(&phase.PrepareHost{phase.Analytics{"Hosts Prepared", nil}})
	phaseManager.AddPhase(&phase.InstallEngine{phase.Analytics{"Engine Installed", nil}})
	phaseManager.AddPhase(&phase.PullImages{phase.Analytics{"Images Pulled", nil}})
	phaseManager.AddPhase(&phase.InitSwarm{phase.Analytics{"Swarm Initialized", nil}})
	phaseManager.AddPhase(&phase.InstallUCP{phase.Analytics{"UPC Installed", nil}})
	phaseManager.AddPhase(&phase.UpgradeUcp{phase.Analytics{"UCP Upgraded", nil}})
	phaseManager.AddPhase(&phase.JoinManagers{phase.Analytics{"Managers Joined", nil}})
	phaseManager.AddPhase(&phase.JoinWorkers{phase.Analytics{"Workers Joined", nil}})
	phaseManager.AddPhase(&phase.SaveState{phase.Analytics{"Local State Saved", nil}})
	phaseManager.AddPhase(&phase.Disconnect{phase.Analytics{"SSH Connections Disconnected", nil}})

	phaseErr := phaseManager.Run()
	if phaseErr != nil {
		return phaseErr
	}
	props := analytics.NewAnalyticsEventProperties()

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
