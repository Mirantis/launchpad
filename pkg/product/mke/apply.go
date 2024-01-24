package mke

import (
	"crypto/sha1" //nolint:gosec // sha1 is used for simple analytics id generation
	"fmt"

	event "gopkg.in/segmentio/analytics-go.v3"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	mke "github.com/Mirantis/mcc/pkg/product/mke/phase"
)

// Apply - installs Docker Enterprise (MKE, MSR, MCR) on the hosts that are
// defined in the config.  Since MSR3 requires a different set of phases than
// MSR, we need to determine which phases to run based on the MSR version.
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
	)

	// Determine which MSR phases to run based on the MSR version.
	switch p.ClusterConfig.Spec.MSR.MajorVersion() {
	case 2:
		phaseManager.AddPhases(
			&mke.PullMSRImages{},
			&mke.ValidateMKEHealth{},
			&mke.InstallMSR{},
			&mke.UpgradeMSR{},
			&mke.JoinMSRReplicas{},
		)
	case 3:
		phaseManager.AddPhases(
			&mke.ValidateMKEHealth{},
			&mke.ConfigureDepsMSR3{},
			&mke.ConfigureStorageProvisioner{},
			&mke.InstallOrUpgradeMSR3{},
		)
	default:
		return fmt.Errorf("unsupported MSR version: %s", p.ClusterConfig.Spec.MSR.Version)
	}

	// Add the remaining phases that run regardless of MSR version.
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
