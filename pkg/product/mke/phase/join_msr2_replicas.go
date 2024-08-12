package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr/msr2"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/alessio/shellescape"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// JoinMSRReplicas phase implementation.
type JoinMSR2Replicas struct {
	phase.Analytics
	phase.HostSelectPhase
	phase.CleanupDisabling
}

// HostFilterFunc returns true for hosts that don't have MSR configured.
func (p *JoinMSR2Replicas) HostFilterFunc(h *api.Host) bool {
	return h.MSR2Metadata == nil || !h.MSR2Metadata.Installed
}

// Prepare collects the hosts.
func (p *JoinMSR2Replicas) Prepare(config interface{}) error {
	cfg, ok := config.(*api.ClusterConfig)
	if !ok {
		return errInvalidConfig
	}
	p.Config = cfg
	if !p.Config.Spec.ContainsMSR() {
		return nil
	}
	log.Debugf("collecting hosts for phase %s", p.Title())
	msrHosts := p.Config.Spec.MSR2s()
	hosts := msrHosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase.
func (p *JoinMSR2Replicas) Title() string {
	return "Join MSR2 Replicas"
}

// ShouldRun should return true only when there is a configured installation.
func (p *JoinMSR2Replicas) ShouldRun() bool {
	return p.Config.Spec.ContainsMSR2()
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinMSR2Replicas) Run() error {
	msrLeader := p.Config.Spec.MSR2Leader()
	mkeFlags := msr2.BuildMKEFlags(p.Config)

	for _, h := range p.Hosts {
		// Iterate through the msrs and determine which have MSR installed
		// on them, if one is found which is not yet in the cluster, perform
		// a join against msrLeader
		if h.MSR2Metadata != nil && h.MSR2Metadata.Installed {
			log.Infof("%s: already a MSR node", h)
			continue
		}

		// Run the join with the appropriate flags taken from the install spec
		runFlags := common.Flags{"-i"}
		if !p.CleanupDisabled() {
			runFlags.Add("--rm")
		}

		if msrLeader.Configurer.SELinuxEnabled(h) {
			runFlags.Add("--security-opt label=disable")
		}
		joinFlags := common.Flags{}
		redacts := []string{}
		joinFlags.Add(fmt.Sprintf("--ucp-node %s", h.Metadata.Hostname))
		joinFlags.Add(fmt.Sprintf("--existing-replica-id %s", msrLeader.MSR2Metadata.ReplicaID))
		joinFlags.MergeOverwrite(mkeFlags)
		// We can't just append the installFlags to joinFlags because they
		// differ, so we have to selectively pluck the ones that are shared
		for _, f := range msr2.PluckSharedInstallFlags(p.Config.Spec.MSR2.InstallFlags, msr2.SharedInstallJoinFlags) {
			joinFlags.AddOrReplace(f)
		}
		if p.Config.Spec.MKE.CACertData != "" {
			escaped := shellescape.Quote(p.Config.Spec.MKE.CACertData)
			joinFlags.AddOrReplace(fmt.Sprintf("--ucp-ca %s", escaped))
			redacts = append(redacts, escaped)
		}
		if h.MSR2Metadata != nil && h.MSR2Metadata.ReplicaID != "" {
			log.Infof("%s: joining MSR replica to cluster with replica id: %s", h, h.MSR2Metadata.ReplicaID)
			joinFlags.AddOrReplace(fmt.Sprintf("--replica-id %s", h.MSR2Metadata.ReplicaID))
		} else {
			log.Infof("%s: joining MSR replica to cluster", h)
		}

		joinCmd := msrLeader.Configurer.DockerCommandf("run %s %s join %s", runFlags.Join(), msrLeader.MSR2Metadata.InstalledBootstrapImage, joinFlags.Join())
		err := msrLeader.Exec(joinCmd, exec.StreamOutput(), exec.RedactString(redacts...))
		if err != nil {
			return fmt.Errorf("%s: failed to run MSR join: %w", h, err)
		}
	}
	return nil
}
