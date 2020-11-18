package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	log "github.com/sirupsen/logrus"
)

// InstallMSR is the phase implementation for running the actual MSR installer
// bootstrap
type InstallMSR struct {
	phase.Analytics
	MSRPhase

	SkipCleanup bool
}

// Title prints the phase title
func (p *InstallMSR) Title() string {
	return "Install MSR components"
}

// Run the installer container
func (p *InstallMSR) Run() error {
	msrLeader := p.Config.Spec.MSRLeader()

	err := p.Config.Spec.CheckMKEHealthRemote(msrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to health check mke, try to set `--ucp-url` installFlag and check connectivity", msrLeader)
	}

	if !p.SkipCleanup {
		defer func() {
			if err != nil {
				log.Println("Cleaning-up")
				if cleanupErr := msr.Destroy(msrLeader); cleanupErr != nil {
					log.Warnln("Error while cleaning-up resources")
					log.Debugf("Cleanup resources error: %s", err)
				}
			}
		}()
	}

	p.EventProperties = map[string]interface{}{
		"msr_version": p.Config.Spec.MSR.Version,
	}

	if p.Config.Spec.MSR.Metadata.Installed {
		log.Infof("%s: MSR already installed at version %s, not running installer", msrLeader, p.Config.Spec.MSR.Metadata.InstalledVersion)
		return nil
	}

	image := p.Config.Spec.MSR.GetBootstrapperImage()
	runFlags := []string{"--rm", "-i"}
	if msrLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	installFlags := p.Config.Spec.MSR.InstallFlags

	// Configure the mkeFlags from existing MKEConfig
	mkeFlags := msr.BuildMKEFlags(p.Config)
	// Conduct the install passing the --ucp-node flag for the host provided in
	// msrLeader.
	mkeFlags = append(mkeFlags, fmt.Sprintf("--ucp-node %s", msrLeader.Metadata.LongHostname))

	installFlags = append(installFlags, mkeFlags...)
	installCmd := msrLeader.Configurer.DockerCommandf("run %s %s install %s", strings.Join(runFlags, " "), image, strings.Join(installFlags, " "))
	err = msrLeader.Exec(installCmd, exec.StreamOutput(), exec.RedactString(installFlags.GetValue("--ucp-username"), installFlags.GetValue("--ucp-password")))
	if err != nil {
		return fmt.Errorf("%s: failed to run MSR installer: %s", msrLeader, err.Error())
	}

	msrMeta, err := msr.CollectFacts(msrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing MSR details: %s", msrLeader, err)
	}
	p.Config.Spec.MSR.Metadata = msrMeta
	return nil
}
