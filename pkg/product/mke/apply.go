package mke

import (
	"crypto/sha1" //nolint:gosec // sha1 is used for simple analytics id generation
	"fmt"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	mke "github.com/Mirantis/mcc/pkg/product/mke/phase"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply - installs Docker Enterprise (MKE, MSR, MCR) on the hosts that are defined in the config.
func (p *MKE) Apply(disableCleanup, force bool, concurrency int, forceUpgrade bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup

	phaseManager.AddPhases(
		&mke.UpgradeCheck{},
		&common.Connect{},
		&mke.DetectOS{},
		&mke.GatherFacts{},
		&mke.ValidateFacts{Force: force},
		&mke.ValidateHosts{},
		&mke.DownloadInstaller{},
		&common.RunHooks{Stage: "before", Action: "apply"},
		&mke.PrepareHost{},

		// begin mcr/mke phases
		&mke.ConfigureMCR{},
		&mke.InstallMCR{},
		&mke.UpgradeMCR{Concurrency: concurrency, ForceUpgrade: forceUpgrade},
		&mke.RestartMCR{},
		&mke.LoadImages{},
		&mke.AuthenticateDocker{},
		&mke.PullMKEImages{},
		&mke.InitSwarm{},
		&mke.InstallMKECerts{},
		&mke.InstallMKE{},
		&mke.UpgradeMKE{},
		&mke.JoinManagers{},
		&mke.JoinWorkers{},
		&mke.ValidateMKEHealth{},
	)

	// Determine which MSR phases to run based on the MSR spec provided in the
	// config.
	if p.ClusterConfig.Spec.ContainsMSR2() {
		phaseManager.AddPhases(
			&mke.PullMSR2Images{},
			&mke.InstallMSR2{},
			&mke.UpgradeMSR2{},
			&mke.JoinMSR2Replicas{},
		)
	}

	if p.ClusterConfig.Spec.ContainsMSR3() {
		phaseManager.AddPhases(
			&mke.ConfigureDepsMSR3{},
			&mke.ConfigureStorageProvisioner{},
			&mke.InstallOrUpgradeMSR3{},
		)
	}

	// Add the remaining phases that run after the MKE and MSR phases.
	phaseManager.AddPhases(
		&mke.LabelNodes{},
		&mke.RemoveNodes{},
		&common.RunHooks{Stage: "after", Action: "apply"},
		&common.Disconnect{},
		&mke.Info{},
	)

	if err := phaseManager.Run(); err != nil {
		return fmt.Errorf("failed to apply MKE: %w", err)
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

	clusterID := p.ClusterConfig.Spec.MCR.Metadata.ClusterID

	props := event.Properties{
		"kind":            p.ClusterConfig.Kind,
		"api_version":     p.ClusterConfig.APIVersion,
		"hosts":           len(p.ClusterConfig.Spec.Hosts),
		"managers":        len(p.ClusterConfig.Spec.Managers()),
		"dtrs":            len(p.ClusterConfig.Spec.MSR2s()),
		"linux_workers":   linuxWorkersCount,
		"windows_workers": windowsWorkersCount,
		"engine_version":  p.ClusterConfig.Spec.MCR.Version,
		"cluster_id":      clusterID,
		// send mke analytics user id as ucp_instance_id property
		"ucp_instance_id": fmt.Sprintf("%x", sha1.Sum([]byte(clusterID))), //nolint:gosec // sha1 is used for simple analytics id generation
	}

	analytics.TrackEvent("Cluster Installed", props)

	return nil
}
