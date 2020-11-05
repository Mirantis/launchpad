package docker_enterprise

import (
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply - installs Docker Enterprise (UCP, DTR, Engine) on the hosts that are defined in the config
func (p *DockerEnterprise) Apply() error {
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
	logFile, err = addFileLogger(p.ClusterConfig.Metadata.Name)
	if err != nil {
		return err
	}

	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = p.SkipCleanup

	phaseManager.AddPhases(
		&phase.Connect{},
		&phase.GatherFacts{},
		&phase.ValidateFacts{},
		&phase.ValidateHosts{Debug: p.Debug},
		&phase.DownloadInstaller{},
		&phase.RunHooks{Stage: "Before", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.Before }},
		&phase.PrepareHost{},
		&phase.InstallEngine{},
		&phase.LoadImages{},
		&phase.PullUCPImages{},
		&phase.InitSwarm{},
		&phase.InstallUCP{SkipCleanup: phaseManager.SkipCleanup},
		&phase.UpgradeUcp{},
		&phase.JoinManagers{},
		&phase.JoinWorkers{},
		// begin DTR phases
		&phase.PullDTRImages{},
		&phase.ValidateUcpHealth{},
		&phase.InstallDtr{SkipCleanup: phaseManager.SkipCleanup},
		&phase.UpgradeDtr{},
		&phase.JoinDtrReplicas{},
		// end DTR phases
		&phase.LabelNodes{},
		&phase.RemoveNodes{},
		&phase.RunHooks{Stage: "After", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.After }},
		&phase.Disconnect{},
		&phase.Info{},
	)

	if err = phaseManager.Run(); err != nil {
		return err
	}

	windowsWorkersCount := 0
	linuxWorkersCount := 0
	for _, h := range p.ClusterConfig.Spec.Workers() {
		if h.IsWindows() {
			windowsWorkersCount++
		} else {
			linuxWorkersCount++
		}
	}
	clusterID := p.ClusterConfig.Spec.Ucp.Metadata.ClusterID
	props := event.Properties{
		"kind":            p.ClusterConfig.Kind,
		"api_version":     p.ClusterConfig.APIVersion,
		"hosts":           len(p.ClusterConfig.Spec.Hosts),
		"managers":        len(p.ClusterConfig.Spec.Managers()),
		"dtrs":            len(p.ClusterConfig.Spec.Dtrs()),
		"linux_workers":   linuxWorkersCount,
		"windows_workers": windowsWorkersCount,
		"engine_version":  p.ClusterConfig.Spec.Engine.Version,
		"cluster_id":      clusterID,
		// send ucp analytics user id as ucp_instance_id property
		"ucp_instance_id": fmt.Sprintf("%x", sha1.Sum([]byte(clusterID))),
	}

	if err := analytics.TrackEvent("Cluster Installed", props); err != nil {
		log.Warnf("tracking failed: %v", err)
	}

	return nil
}
