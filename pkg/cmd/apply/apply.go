package apply

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/util"
	"github.com/Mirantis/mcc/version"
	"github.com/mattn/go-isatty"
	"github.com/mitchellh/go-homedir"

	mcclog "github.com/Mirantis/mcc/pkg/log"
	log "github.com/sirupsen/logrus"
)

// Apply ...
func Apply(configFile string, prune bool) error {
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

	if err = analytics.RequireRegisteredUser(); err != nil {
		return err
	}
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

	// Add logger to dump all log levels to file
	logFile, err = addFileLogger(clusterConfig.Metadata.Name)
	if err != nil {
		return err
	}

	phaseManager := phase.NewManager(&clusterConfig)
	phaseManager.AddPhase(&phase.Connect{})
	phaseManager.AddPhase(&phase.GatherFacts{})
	phaseManager.AddPhase(&phase.ValidateHosts{})
	phaseManager.AddPhase(&phase.DownloadInstaller{})
	phaseManager.AddPhase(&phase.PrepareHost{})
	phaseManager.AddPhase(&phase.InstallEngine{})
	phaseManager.AddPhase(&phase.PullImages{})
	phaseManager.AddPhase(&phase.InitSwarm{})
	phaseManager.AddPhase(&phase.InstallUCP{})
	phaseManager.AddPhase(&phase.UpgradeUcp{})
	phaseManager.AddPhase(&phase.JoinManagers{})
	phaseManager.AddPhase(&phase.JoinWorkers{})
	phaseManager.AddPhase(&phase.LabelNodes{})
	if prune {
		phaseManager.AddPhase(&phase.RemoveNodes{})
	}
	phaseManager.AddPhase(&phase.Disconnect{})
	phaseManager.AddPhase(&phase.UcpInfo{})

	if err = phaseManager.Run(); err != nil {
		return err
	}

	props := analytics.NewAnalyticsEventProperties()
	props["kind"] = clusterConfig.Kind
	props["api_version"] = clusterConfig.APIVersion
	props["hosts"] = len(clusterConfig.Spec.Hosts)
	props["managers"] = len(clusterConfig.Spec.Managers())
	linuxWorkersCount := 0
	windowsWorkersCount := 0
	for _, h := range clusterConfig.Spec.Workers() {
		if h.IsWindows() {
			windowsWorkersCount++
		} else {
			linuxWorkersCount++
		}
	}
	props["linux_workers"] = linuxWorkersCount
	props["windows_workers"] = windowsWorkersCount
	props["engine_version"] = clusterConfig.Spec.Engine.Version
	clusterID := clusterConfig.Spec.Ucp.Metadata.ClusterID
	props["cluster_id"] = clusterID
	// send ucp analytics user id as ucp_instance_id property
	ucpInstanceID := fmt.Sprintf("%x", sha1.Sum([]byte(clusterID)))
	props["ucp_instance_id"] = ucpInstanceID
	analytics.TrackEvent("Cluster Installed", props)
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
