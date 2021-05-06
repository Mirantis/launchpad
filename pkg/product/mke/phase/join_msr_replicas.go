package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// JoinMSRReplicas phase implementation
type JoinMSRReplicas struct {
	phase.Analytics
	phase.HostSelectPhase
	phase.CleanupDisabling
}

// HostFilterFunc returns true for hosts that don't have MSR configured
func (p *JoinMSRReplicas) HostFilterFunc(h *api.Host) bool {
	return h.MSRMetadata == nil || !h.MSRMetadata.Installed
}

// Prepare collects the hosts
func (p *JoinMSRReplicas) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	if !p.Config.Spec.ContainsMSR() {
		return nil
	}
	log.Debugf("collecting hosts for phase %s", p.Title())
	msrHosts := p.Config.Spec.MSRs()
	hosts := msrHosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *JoinMSRReplicas) Title() string {
	return "Join MSR Replicas"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinMSRReplicas) Run() error {
	msrLeader := p.Config.Spec.MSRLeader()
	mkeFlags := msr.BuildMKEFlags(p.Config)

	for _, h := range p.Hosts {
		// Iterate through the msrs and determine which have MSR installed
		// on them, if one is found which is not yet in the cluster, perform
		// a join against msrLeader
		if h.MSRMetadata != nil && h.MSRMetadata.Installed {
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
		joinFlags.Add(fmt.Sprintf("--ucp-node %s", h.Metadata.LongHostname))
		joinFlags.Add(fmt.Sprintf("--existing-replica-id %s", msrLeader.MSRMetadata.ReplicaID))
		joinFlags.MergeOverwrite(mkeFlags)
		// We can't just append the installFlags to joinFlags because they
		// differ, so we have to selectively pluck the ones that are shared
		for _, f := range msr.PluckSharedInstallFlags(p.Config.Spec.MSR.InstallFlags, msr.SharedInstallJoinFlags) {
			joinFlags.AddOrReplace(f)
		}
		if h.MSRMetadata != nil && h.MSRMetadata.ReplicaID != "" {
			log.Infof("%s: joining MSR replica to cluster with with replica id: %s", h, h.MSRMetadata.ReplicaID)
			joinFlags.AddOrReplace(fmt.Sprintf("--replica-id %s", h.MSRMetadata.ReplicaID))
		} else {
			log.Infof("%s: joining MSR replica to cluster", h)
		}

		joinCmd := msrLeader.Configurer.DockerCommandf("run %s %s join %s", runFlags.Join(), msrLeader.MSRMetadata.InstalledBootstrapImage, joinFlags.Join())
		err := msrLeader.Exec(joinCmd, exec.StreamOutput())
		if err != nil {
			return fmt.Errorf("%s: failed to run MSR join: %s", h, err.Error())
		}
	}
	return nil
}
