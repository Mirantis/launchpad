package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	log "github.com/sirupsen/logrus"
)

// JoinMSRReplicas phase implementation
type JoinMSRReplicas struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that have non-empty list of hooks returned by the StepListFunc
func (p *JoinMSRReplicas) HostFilterFunc(h *api.Host) bool {
	return h.MSRMetadata == nil || !h.MSRMetadata.Installed
}

// Prepare collects the hosts
func (p *JoinMSRReplicas) Prepare(config *api.ClusterConfig) error {
	if !config.Spec.ContainsMSR() {
		return nil
	}
	p.Config = config
	log.Debugf("collecting hosts for phase %s", p.Title())
	msrHosts := config.Spec.MSRs()
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
		if h.MSRMetadata.Installed {
			log.Infof("%s: already a MSR node", h)
			continue
		}

		// Run the join with the appropriate flags taken from the install spec
		runFlags := common.Flags{"--rm", "-i"}
		if msrLeader.Configurer.SELinuxEnabled() {
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
