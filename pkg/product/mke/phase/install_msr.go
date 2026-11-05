package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/msr"
	"github.com/Mirantis/launchpad/pkg/phase"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/k0sproject/rig/v2/cmd"
	log "github.com/sirupsen/logrus"
)

// InstallMSR is the phase implementation for running the actual MSR installer
// bootstrap.
type InstallMSR struct {
	phase.Analytics
	phase.CleanupDisabling
	MSRPhase

	leader *mkeconfig.Host
}

// Title prints the phase title.
func (p *InstallMSR) Title() string {
	return "Install MSR components"
}

// ShouldRun should return true only when there is an installation to be performed.
func (p *InstallMSR) ShouldRun() bool {
	p.leader = p.Config.Spec.MSRLeader()
	return p.Config.Spec.ContainsMSR() && (p.leader.MSRMetadata == nil || !p.leader.MSRMetadata.Installed)
}

// Run the installer container.
func (p *InstallMSR) Run() error {
	h := p.leader
	if h.MSRMetadata == nil {
		h.MSRMetadata = &mkeconfig.MSRMetadata{}
	}

	p.EventProperties = map[string]interface{}{
		"msr_version": p.Config.Spec.MSR.Version,
	}

	installFlags := p.Config.Spec.MSR.InstallFlags
	redacts := []string{installFlags.GetValue("--ucp-username"), installFlags.GetValue("--ucp-password")}

	// Configure the mkeFlags from existing MKEConfig
	mkeFlags := msr.BuildMKEFlags(p.Config)
	// Conduct the install passing the --ucp-node flag for the host provided in
	// msrLeader.
	mkeFlags.AddOrReplace(fmt.Sprintf("--ucp-node %s", h.Metadata.Hostname))

	installFlags.Merge(mkeFlags)

	// The TLS material is PEM, multi-line, and secret. It goes to the
	// installer as environment values staged in mode-0600 files on the host
	// by pkg/util/secretenv, never on the command line, so no shell quoting
	// of a private key is needed and nothing records it.
	secretEnv := map[string]string{
		"MSR_CA":   p.Config.Spec.MSR.CACertData,
		"MSR_CERT": p.Config.Spec.MSR.CertData,
		"MSR_KEY":  p.Config.Spec.MSR.KeyData,
		"UCP_CA":   p.Config.Spec.MKE.CACertData,
	}

	if h.MSRMetadata.ReplicaID != "" {
		log.Infof("%s: installing MSR with replica id %s", h, h.MSRMetadata.ReplicaID)
		installFlags.AddOrReplace(fmt.Sprintf("--replica-id %s", h.MSRMetadata.ReplicaID))
	} else {
		log.Infof("%s: installing MSR version %s", h, p.Config.Spec.MSR.Version)
	}

	execOpts := make([]cmd.ExecOption, 0, len(redacts))
	for _, r := range redacts {
		execOpts = append(execOpts, cmd.Redact(r))
	}
	if _, err := msr.Bootstrap("install", *p.Config, msr.BootstrapOptions{OperationFlags: installFlags, CleanupDisabled: p.CleanupDisabled(), SecretEnv: secretEnv, ExecOptions: execOpts}); err != nil {
		return fmt.Errorf("%s: failed to run MSR installer: %w", h, err)
	}

	msrMeta, err := msr.CollectFacts(h)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MSR details: %w", h, err)
	}
	h.MSRMetadata = msrMeta
	return nil
}

// CleanUp removes remnants of MSR after a failed installation.
func (p *InstallMSR) CleanUp() {
	log.Infof("Cleaning up for '%s'", p.Title())
	if err := msr.Destroy(p.leader, p.Config); err != nil {
		log.Warnf("Error while cleaning-up resources for '%s': %s", p.Title(), err.Error())
	}
}
