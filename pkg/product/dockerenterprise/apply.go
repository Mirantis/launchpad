package dockerenterprise

import (
	"crypto/sha1"
	"fmt"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	de "github.com/Mirantis/mcc/pkg/product/dockerenterprise/phase"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply - installs Docker Enterprise (UCP, DTR, Engine) on the hosts that are defined in the config
func (p *DockerEnterprise) Apply(disableCleanup, force bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup

	phaseManager.AddPhases(
		&common.Connect{},
		&de.GatherFacts{},
		&de.ValidateFacts{Force: force},
		&de.ValidateHosts{},
		&de.DownloadInstaller{},
		&common.RunHooks{Stage: "Before", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.Before }},
		&de.PrepareHost{},
		&de.InstallEngine{},
		&de.LoadImages{},
		&de.PullUCPImages{},
		&de.InitSwarm{},
		&de.InstallUCP{SkipCleanup: disableCleanup},
		&de.UpgradeUcp{},
		&de.JoinManagers{},
		&de.JoinWorkers{},

		// begin DTR phases
		&de.PullDTRImages{},
		&de.ValidateUcpHealth{},
		&de.InstallDtr{SkipCleanup: disableCleanup},
		&de.UpgradeDtr{},
		&de.JoinDtrReplicas{},
		// end DTR phases

		&de.LabelNodes{},
		&de.RemoveNodes{},
		&common.RunHooks{Stage: "After", Action: "Apply", StepListFunc: func(h *api.Host) *[]string { return h.Hooks.Apply.After }},
		&common.Disconnect{},
		&de.Info{},
	)

	if err := phaseManager.Run(); err != nil {
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
