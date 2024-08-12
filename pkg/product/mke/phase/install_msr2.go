package phase

import (
	"fmt"

	msr "github.com/Mirantis/mcc/pkg/msr/msr2"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/alessio/shellescape"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// InstallMSR is the phase implementation for running the actual MSR installer
// bootstrap.
type InstallMSR2 struct {
	phase.Analytics
	phase.CleanupDisabling
	phase.BasicPhase

	leader *api.Host
}

// Title prints the phase title.
func (p *InstallMSR2) Title() string {
	return "Install MSR components"
}

// ShouldRun should return true only when there is an installation to be
// performed.
func (p *InstallMSR2) ShouldRun() bool {
	if !p.Config.Spec.ContainsMSR2() {
		return false
	}
	p.leader = p.Config.Spec.MSR2Leader()
	return p.Config.Spec.ContainsMSR2() && (p.leader.MSR2Metadata == nil || !p.leader.MSR2Metadata.Installed)
}

// Run the installer container.
func (p *InstallMSR2) Run() error {
	h := p.leader

	if h.MSR2Metadata == nil {
		h.MSR2Metadata = &api.MSR2Metadata{}
	}

	managers := p.Config.Spec.Managers()

	err := p.Config.Spec.CheckMKEHealthRemote(managers)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity: %w", h, err)
	}

	p.EventProperties = map[string]interface{}{
		"msr2_version": p.Config.Spec.MSR2.Version,
	}

	image := p.Config.Spec.MSR2.GetBootstrapperImage()

	runFlags := common.Flags{"-i"}
	if !p.CleanupDisabled() {
		runFlags.Add("--rm")
	}

	if h.Configurer.SELinuxEnabled(h) {
		runFlags.Add("--security-opt label=disable")
	}
	installFlags := p.Config.Spec.MSR2.InstallFlags
	redacts := []string{installFlags.GetValue("--ucp-username"), installFlags.GetValue("--ucp-password")}

	// Configure the mkeFlags from existing MKEConfig
	mkeFlags := msr.BuildMKEFlags(p.Config)
	// Conduct the install passing the --ucp-node flag for the host provided in
	// msrLeader.
	mkeFlags.AddOrReplace(fmt.Sprintf("--ucp-node %s", h.Metadata.Hostname))

	installFlags.Merge(mkeFlags)

	if p.Config.Spec.MSR2.CACertData != "" {
		escaped := shellescape.Quote(p.Config.Spec.MSR2.CACertData)
		installFlags.AddOrReplace(fmt.Sprintf("--dtr-ca %s", escaped))
		redacts = append(redacts, escaped)
	}

	if p.Config.Spec.MSR2.CertData != "" {
		escaped := shellescape.Quote(p.Config.Spec.MSR2.CertData)
		installFlags.AddOrReplace(fmt.Sprintf("--dtr-cert %s", escaped))
		redacts = append(redacts, escaped)
	}

	if p.Config.Spec.MSR2.KeyData != "" {
		escaped := shellescape.Quote(p.Config.Spec.MSR2.KeyData)
		installFlags.AddOrReplace(fmt.Sprintf("--dtr-key %s", escaped))
		redacts = append(redacts, escaped)
	}

	if p.Config.Spec.MKE.CACertData != "" {
		escaped := shellescape.Quote(p.Config.Spec.MKE.CACertData)
		installFlags.AddOrReplace(fmt.Sprintf("--ucp-ca %s", escaped))
		redacts = append(redacts, escaped)
	}

	if h.MSR2Metadata.ReplicaID != "" {
		log.Infof("%s: installing MSR2 with replica id %s", h, h.MSR2Metadata.ReplicaID)
		installFlags.AddOrReplace(fmt.Sprintf("--replica-id %s", h.MSR2Metadata.ReplicaID))
	} else {
		log.Infof("%s: installing MSR2 version %s", h, p.Config.Spec.MSR2.Version)
	}

	installCmd := h.Configurer.DockerCommandf("run %s %s install %s", runFlags.Join(), image, installFlags.Join())
	err = h.Exec(installCmd, exec.StreamOutput(), exec.RedactString(redacts...))
	if err != nil {
		return fmt.Errorf("%s: failed to run MSR installer: %w", h, err)
	}

	msrMeta, err := msr.CollectFacts(h)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MSR details: %w", h, err)
	}
	h.MSR2Metadata = msrMeta
	return nil
}

// CleanUp removes remnants of MSR2 after a failed installation.
func (p *InstallMSR2) CleanUp() {
	log.Infof("Cleaning up for '%s'", p.Title())
	if err := msr.Destroy(p.leader, p.Config); err != nil {
		log.Warnf("Error while cleaning-up resources for '%s': %s", p.Title(), err.Error())
	}
}
