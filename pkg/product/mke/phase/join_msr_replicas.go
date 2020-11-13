package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// JoinMSRReplicas phase implementation
type JoinMSRReplicas struct {
	phase.Analytics
	MSRPhase
}

// Title for the phase
func (p *JoinMSRReplicas) Title() string {
	return "Join MSR Replicas"
}

// Run joins all the workers nodes to swarm if not already part of it.
func (p *JoinMSRReplicas) Run() error {
	msrLeader := p.Config.Spec.MSRLeader()
	mkeFlags := msr.BuildMKEFlags(p.Config)
	sequentialInt := 0

	for _, d := range p.Config.Spec.MSRs() {
		sequentialInt++
		// Iterate through the msrs and determine which have MSR installed
		// on them, if one is found which is not yet in the cluster, perform
		// a join against msrLeader
		if api.IsMSRInstalled(d) {
			log.Infof("%s: already a MSR node", d)
			continue
		}

		// Run the join with the appropriate flags taken from the install spec
		runFlags := []string{"--rm", "-i"}
		if msrLeader.Configurer.SELinuxEnabled() {
			runFlags = append(runFlags, "--security-opt label=disable")
		}
		joinFlags := []string{
			fmt.Sprintf("--ucp-node %s", d.Metadata.LongHostname),
			fmt.Sprintf("--existing-replica-id %s", p.Config.Spec.MSR.Metadata.MSRLeaderReplicaID),
		}
		if p.Config.Spec.MSR.ReplicaConfig == "sequential" {
			// Assign the appropriate sequential replica value if set
			builtSeqInt := msr.SequentialReplicaID(sequentialInt)
			log.Debugf("Joining replica with sequential replicaID: %s", builtSeqInt)
			joinFlags = append(joinFlags, fmt.Sprintf("--replica-id %s", builtSeqInt))
		}
		joinFlags = append(joinFlags, mkeFlags...)
		// We can't just append the installFlags to joinFlags because they
		// differ, so we have to selectively pluck the ones that are shared
		for _, f := range msr.PluckSharedInstallFlags(p.Config.Spec.MSR.InstallFlags, msr.SharedInstallJoinFlags) {
			joinFlags = append(joinFlags, f)
		}

		joinCmd := msrLeader.Configurer.DockerCommandf("run %s %s join %s", strings.Join(runFlags, " "), p.Config.Spec.MSR.Metadata.InstalledBootstrapImage, strings.Join(joinFlags, " "))
		log.Debugf("%s: Joining MSR replica to cluster", d)
		err := msrLeader.Exec(joinCmd, exec.StreamOutput())
		if err != nil {
			return fmt.Errorf("%s: failed to run MSR join: %s", msrLeader, err.Error())
		}
	}
	return nil
}
