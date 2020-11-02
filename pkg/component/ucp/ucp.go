package ucp

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
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// UCP - Universal Control Plane component
type UCP struct {
	ClusterConfig api.ClusterConfig
	SkipCleanup   bool
	Debug         bool
}

// Apply - installs UCP on the hosts that are defined in the config
func (u UCP) Apply() error {
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

	// Add logger to dump all log levels to file
	logFile, err = addFileLogger(u.ClusterConfig.Metadata.Name)
	if err != nil {
		return err
	}

	dtr := config.ContainsDtr(u.ClusterConfig)

	phaseManager := phase.NewManager(&u.ClusterConfig)
	phaseManager.SkipCleanup = u.SkipCleanup

	phaseManager.AddPhases(&phase.Connect{},
		&phase.GatherFacts{Dtr: dtr},
		&phase.ValidateFacts{},
		&phase.ValidateHosts{Debug: u.Debug},
		&phase.DownloadInstaller{},
		&phase.RunHooks{Stage: "Before", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.Before }},
		&phase.PrepareHost{},
		&phase.InstallEngine{},
		&phase.LoadImages{},
		&phase.PullImages{},
		&phase.InitSwarm{},
		&phase.InstallUCP{SkipCleanup: phaseManager.SkipCleanup},
		&phase.UpgradeUcp{},
		&phase.JoinManagers{},
		&phase.JoinWorkers{})

	// If the clusterConfig contains any of the DTR role then install and
	// upgrade DTR on those specific host roles
	if dtr {
		phaseManager.AddPhases(&phase.PullImages{Dtr: dtr},
			&phase.ValidateUcpHealth{},
			&phase.InstallDtr{SkipCleanup: phaseManager.SkipCleanup},
			&phase.UpgradeDtr{},
			&phase.JoinDtrReplicas{})
	}

	phaseManager.AddPhases(&phase.LabelNodes{},
		&phase.RemoveNodes{},
		&phase.RunHooks{Stage: "After", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.After }},
		&phase.Disconnect{},
		&phase.Info{})

	if err = phaseManager.Run(); err != nil {
		return err
	}

	windowsWorkersCount := 0
	linuxWorkersCount := 0
	for _, h := range u.ClusterConfig.Spec.Workers() {
		if h.IsWindows() {
			windowsWorkersCount++
		} else {
			linuxWorkersCount++
		}
	}
	clusterID := u.ClusterConfig.Spec.Ucp.Metadata.ClusterID
	props := event.Properties{
		"kind":            u.ClusterConfig.Kind,
		"api_version":     u.ClusterConfig.APIVersion,
		"hosts":           len(u.ClusterConfig.Spec.Hosts),
		"managers":        len(u.ClusterConfig.Spec.Managers()),
		"dtrs":            len(u.ClusterConfig.Spec.Dtrs()),
		"linux_workers":   linuxWorkersCount,
		"windows_workers": windowsWorkersCount,
		"engine_version":  u.ClusterConfig.Spec.Engine.Version,
		"cluster_id":      clusterID,
		// send ucp analytics user id as ucp_instance_id property
		"ucp_instance_id": fmt.Sprintf("%x", sha1.Sum([]byte(clusterID))),
	}

	if err := analytics.TrackEvent("Cluster Installed", props); err != nil {
		log.Warnf("tracking failed: %v", err)
	}

	return nil
}

// Reset - reinstall
func (u UCP) Reset() error {
	log.Debugf("loaded cluster cfg: %+v", u.ClusterConfig)

	dtr := config.ContainsDtr(u.ClusterConfig)
	phaseManager := phase.NewManager(&u.ClusterConfig)

	phaseManager.AddPhases(&phase.Connect{}, &phase.GatherFacts{Dtr: dtr},
		&phase.RunHooks{Stage: "Before", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.Before }})
	if dtr {
		phaseManager.AddPhase(&phase.UninstallDTR{})
	}
	phaseManager.AddPhases(&phase.UninstallUCP{},
		&phase.DownloadInstaller{}, &phase.UninstallEngine{},
		&phase.CleanUp{},
		&phase.RunHooks{Stage: "After", Action: "Reset", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Reset.After }},
		&phase.Disconnect{})

	return phaseManager.Run()
}

// Describe - gets information about configured instance
func (u UCP) Describe(reportName string) error {
	var dtr bool
	var ucp bool

	if reportName == "dtr" {
		dtr = true
	}

	if reportName == "ucp" {
		ucp = true
	}

	log.Debugf("loaded cluster cfg: %+v", u.ClusterConfig)

	phaseManager := phase.NewManager(&u.ClusterConfig)
	phaseManager.IgnoreErrors = true

	phaseManager.AddPhases(&phase.Connect{},
		&phase.GatherFacts{Dtr: dtr},
		&phase.Disconnect{} < &phase.Describe{Ucp: ucp, Dtr: dtr})

	return phaseManager.Run()
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
