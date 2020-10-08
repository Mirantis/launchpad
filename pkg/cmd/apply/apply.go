package apply

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/constant"
	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply ...
func Apply(configFile string, prune, force, debug bool) error {
	var (
		logFile *os.File
		err     error
	)
	defer func() {
		// logFile can be nil if error occurred before parsing the config
		if err != nil && logFile != nil {
			log.Infof("See %s for more logs ", logFile.Name())
		}

	}()

	cfgData, err := config.ResolveClusterFile(configFile)
	if err != nil {
		return err
	}
	clusterConfig, err := config.FromYaml(cfgData)
	if err != nil {
		return err
	}

	log.Debugf("validating configuration")
	if err = config.Validate(&clusterConfig); err != nil {
		return err
	}

	if isatty.IsTerminal(os.Stdout.Fd()) {
		os.Stdout.WriteString(util.Logo)
		os.Stdout.WriteString(fmt.Sprintf("   Mirantis Launchpad (c) 2020 Mirantis, Inc.                          v%s\n\n", version.Version))
	}

	// Add logger to dump all log levels to file
	logFile, err = addFileLogger(clusterConfig.Metadata.Name)
	if err != nil {
		return err
	}

	dtr := config.ContainsDtr(clusterConfig)
	clusterConfig.Spec.Metadata.Force = force

	phaseManager := phase.NewManager(&clusterConfig)
	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{Dtr: dtr})
	phaseManager.AddPhase(&phase.ValidateFacts{})
	phaseManager.AddPhase(&phase.ValidateHosts{Debug: debug})
	phaseManager.AddPhase(&phase.DownloadInstaller{})
	phaseManager.AddPhase(&phase.RunHooks{Stage: "Before", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.Before }})
	phaseManager.AddPhase(&phase.PrepareHost{})
	phaseManager.AddPhase(&phase.InstallEngine{})
	phaseManager.AddPhase(&phase.PullImages{})
	phaseManager.AddPhase(&phase.InitSwarm{})
	phaseManager.AddPhase(&phase.InstallUCP{})
	phaseManager.AddPhase(&phase.UpgradeUcp{})
	phaseManager.AddPhase(&phase.JoinManagers{})
	phaseManager.AddPhase(&phase.JoinWorkers{})
	// If the clusterConfig contains any of the DTR role then install and
	// upgrade DTR on those specific host roles
	if dtr {
		phaseManager.AddPhase(&phase.PullImages{Dtr: dtr})
		phaseManager.AddPhase(&phase.ValidateUcpHealth{})
		phaseManager.AddPhase(&phase.InstallDtr{})
		phaseManager.AddPhase(&phase.UpgradeDtr{})
		phaseManager.AddPhase(&phase.JoinDtrReplicas{})
	}
	phaseManager.AddPhase(&phase.LabelNodes{})
	if prune {
		phaseManager.AddPhase(&phase.RemoveNodes{})
	}
	phaseManager.AddPhase(&phase.RunHooks{Stage: "After", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.After }})
	phaseManager.AddPhase(&phase.Disconnect{})
	phaseManager.AddPhase(&phase.Info{})

	if err = phaseManager.Run(); err != nil {
		return err
	}

	windowsWorkersCount := 0
	linuxWorkersCount := 0
	for _, h := range clusterConfig.Spec.Workers() {
		if h.IsWindows() {
			windowsWorkersCount++
		} else {
			linuxWorkersCount++
		}
	}
	clusterID := clusterConfig.Spec.Ucp.Metadata.ClusterID
	props := event.Properties{
		"kind":            clusterConfig.Kind,
		"api_version":     clusterConfig.APIVersion,
		"hosts":           len(clusterConfig.Spec.Hosts),
		"managers":        len(clusterConfig.Spec.Managers()),
		"dtrs":            len(clusterConfig.Spec.Dtrs()),
		"linux_workers":   linuxWorkersCount,
		"windows_workers": windowsWorkersCount,
		"engine_version":  clusterConfig.Spec.Engine.Version,
		"cluster_id":      clusterID,
		// send ucp analytics user id as ucp_instance_id property
		"ucp_instance_id": fmt.Sprintf("%x", sha1.Sum([]byte(clusterID))),
	}

	if err := analytics.TrackEvent("Cluster Installed", props); err != nil {
		log.Warnf("tracking failed: %v", err)
	}

	return nil
}

const fileMode = 0700

func addFileLogger(clusterName string) (*os.File, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, err
	}

	clusterDir := path.Join(home, constant.StateBaseDir, "cluster", clusterName)
	if err := util.EnsureDir(clusterDir); err != nil {
		return nil, fmt.Errorf("error while creating directory for apply logs: %w", err)
	}
	logFileName := path.Join(clusterDir, "apply.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)

	if err != nil {
		return nil, fmt.Errorf("Failed to create apply log at %s: %s", logFileName, err.Error())
	}

	// Send all logs to named file, this ensures we always have debug logs also available when needed
	log.AddHook(mcclog.NewFileHook(logFile))

	return logFile, nil
}
