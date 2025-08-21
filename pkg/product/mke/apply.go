package mke

import (
	"crypto/sha1" //nolint:gosec // sha1 is used for simple analytics id generation
	"fmt"

	"github.com/Mirantis/launchpad/pkg/analytics"
	"github.com/Mirantis/launchpad/pkg/phase"
	common "github.com/Mirantis/launchpad/pkg/product/common/phase"
	mke "github.com/Mirantis/launchpad/pkg/product/mke/phase"
	event "github.com/segmentio/analytics-go/v3"
)

// Apply - installs Docker Enterprise (MKE, MSR, MCR) on the hosts that are defined in the config.
func (p *MKE) Apply(disableCleanup, force bool, concurrency int, forceUpgrade bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup

	phaseManager.AddPhases(
		&mke.UpgradeCheck{},
		&mke.OverrideHostSudo{},
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

		// begin MSR phases
		&mke.PullMSRImages{},
		&mke.ValidateMKEHealth{},
		&mke.InstallMSR{},
		&mke.UpgradeMSR{},
		&mke.JoinMSRReplicas{},
		// end MSR phases

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
	clusterID := p.ClusterConfig.Spec.MKE.Metadata.ClusterID
	props := event.Properties{
		"kind":            p.ClusterConfig.Kind,
		"api_version":     p.ClusterConfig.APIVersion,
		"hosts":           len(p.ClusterConfig.Spec.Hosts),
		"managers":        len(p.ClusterConfig.Spec.Managers()),
		"dtrs":            len(p.ClusterConfig.Spec.MSRs()),
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
