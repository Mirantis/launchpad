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
type JoinMSRReplicas struct {
	phase.Analytics
	phase.HostSelectPhase
	phase.CleanupDisabling
}

// HostFilterFunc returns true for hosts that don't have MSR configured.
func (p *JoinMSRReplicas) HostFilterFunc(h *api.Host) bool {
	return h.MSRMetadata == nil || !h.MSRMetadata.Installed
}

// Prepare collects the hosts.
func (p *JoinMSRReplicas) Prepare(config interface{}) error {
	cfg, ok := config.(*api.ClusterConfig)
	if !ok {
		return errInvalidConfig
	}
	p.Config = cfg
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

// Title for the phase.
func (p *JoinMSRReplicas) Title() string {
	return "Join MSR Replicas"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinMSRReplicas) Run() error {
	msrLeader := p.Config.Spec.MSRLeader()
	mkeFlags := msr2.BuildMKEFlags(p.Config)

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
		redacts := []string{}
		joinFlags.Add(fmt.Sprintf("--ucp-node %s", h.Metadata.Hostname))
		joinFlags.Add(fmt.Sprintf("--existing-replica-id %s", msrLeader.MSRMetadata.MSR2.ReplicaID))
		joinFlags.MergeOverwrite(mkeFlags)
		// We can't just append the installFlags to joinFlags because they
		// differ, so we have to selectively pluck the ones that are shared
		for _, f := range msr2.PluckSharedInstallFlags(p.Config.Spec.MSR.V2.InstallFlags, msr2.SharedInstallJoinFlags) {
			joinFlags.AddOrReplace(f)
		}
		if p.Config.Spec.MKE.CACertData != "" {
			escaped := shellescape.Quote(p.Config.Spec.MKE.CACertData)
			joinFlags.AddOrReplace(fmt.Sprintf("--ucp-ca %s", escaped))
			redacts = append(redacts, escaped)
		}
		if h.MSRMetadata != nil && h.MSRMetadata.MSR2.ReplicaID != "" {
			log.Infof("%s: joining MSR replica to cluster with replica id: %s", h, h.MSRMetadata.MSR2.ReplicaID)
			joinFlags.AddOrReplace(fmt.Sprintf("--replica-id %s", h.MSRMetadata.MSR2.ReplicaID))
		} else {
			log.Infof("%s: joining MSR replica to cluster", h)
		}

		joinCmd := msrLeader.Configurer.DockerCommandf("run %s %s join %s", runFlags.Join(), msrLeader.MSRMetadata.MSR2.InstalledBootstrapImage, joinFlags.Join())
		err := msrLeader.Exec(joinCmd, exec.StreamOutput(), exec.RedactString(redacts...))
		if err != nil {
			return fmt.Errorf("%s: failed to run MSR join: %w", h, err)
		}
	}
	return nil
}
