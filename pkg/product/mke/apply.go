package mke

import (
	"crypto/sha1"
	"fmt"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	mke "github.com/Mirantis/mcc/pkg/product/mke/phase"
	log "github.com/sirupsen/logrus"
	event "gopkg.in/segmentio/analytics-go.v3"
)

// Apply - installs Docker Enterprise (MKE, MSR, Engine) on the hosts that are defined in the config
func (p *MKE) Apply(disableCleanup, force bool) error {
	phaseManager := phase.NewManager(&p.ClusterConfig)
	phaseManager.SkipCleanup = disableCleanup

	phaseManager.AddPhases(
		&common.Connect{},
		&mke.GatherFacts{},
		&mke.ValidateFacts{Force: force},
		&mke.ValidateHosts{},
		&mke.DownloadInstaller{},
		&common.RunHooks{Stage: "Before", Action: "Apply", StepListFunc: func(h *api.Host) *[]string {
			if h.Hooks == nil || h.Hooks.Apply == nil || h.Hooks.Apply.Before == nil {
				return &[]string{}
			}
			return h.Hooks.Apply.Before
		}},
		&mke.PrepareHost{},
		&mke.ConfigureEngine{},
		&mke.InstallEngine{},
		&mke.UpgradeEngine{},
		&mke.RestartEngine{},
		&mke.LoadImages{},
		&mke.AuthenticateDocker{},
		&mke.PullMKEImages{},
		&mke.InitSwarm{},
		&mke.InstallMKE{SkipCleanup: disableCleanup},
		&mke.UpgradeMKE{},
		&mke.JoinManagers{},
		&mke.JoinWorkers{},

		// begin MSR phases
		&mke.PullMSRImages{},
		&mke.ValidateMKEHealth{},
		&mke.InstallMSR{SkipCleanup: disableCleanup},
		&mke.UpgradeMSR{},
		&mke.JoinMSRReplicas{},
		// end MSR phases

		&mke.LabelNodes{},
		&mke.RemoveNodes{},
		&common.RunHooks{Stage: "After", Action: "Apply", StepListFunc: func(h *api.Host) *[]string {
			if h.Hooks == nil || h.Hooks.Apply == nil || h.Hooks.Apply.After == nil {
				return &[]string{}
			}
			return h.Hooks.Apply.After
		}},
		&common.Disconnect{},
		&mke.Info{},
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
	clusterID := p.ClusterConfig.Spec.MKE.Metadata.ClusterID
	props := event.Properties{
		"kind":            p.ClusterConfig.Kind,
		"api_version":     p.ClusterConfig.APIVersion,
		"hosts":           len(p.ClusterConfig.Spec.Hosts),
		"managers":        len(p.ClusterConfig.Spec.Managers()),
		"dtrs":            len(p.ClusterConfig.Spec.MSRs()),
		"linux_workers":   linuxWorkersCount,
		"windows_workers": windowsWorkersCount,
		"engine_version":  p.ClusterConfig.Spec.Engine.Version,
		"cluster_id":      clusterID,
		// send mke analytics user id as ucp_instance_id property
		"ucp_instance_id": fmt.Sprintf("%x", sha1.Sum([]byte(clusterID))),
	}

	if err := analytics.TrackEvent("Cluster Installed", props); err != nil {
		log.Warnf("tracking failed: %v", err)
	}

	return nil
}
