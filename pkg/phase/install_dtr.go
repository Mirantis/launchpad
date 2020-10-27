package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/dtr"
	log "github.com/sirupsen/logrus"
)

// InstallDtr is the phase implementation for running the actual DTR installer
// bootstrap
type InstallDtr struct {
	Analytics
	BasicPhase
}

// Title prints the phase title
func (p *InstallDtr) Title() string {
	return "Install DTR components"
}

// Run the installer container
func (p *InstallDtr) Run() (err error) {
	dtrLeader := p.config.Spec.DtrLeader()

	defer func() {
		if err != nil {
			log.Println("Cleaning-up")
			if cleanupErr := dtr.Destroy(dtrLeader); cleanupErr != nil {
				log.Warnln("Error while cleaning-up resources")
				log.Debugf("Cleanup resources error: %s", err)
			}
		}
	}()

	p.EventProperties = map[string]interface{}{
		"dtr_version": p.config.Spec.Dtr.Version,
	}

	if p.config.Spec.Dtr.Metadata.Installed {
		log.Infof("%s: DTR already installed at version %s, not running installer", dtrLeader.Address, p.config.Spec.Dtr.Metadata.InstalledVersion)
		return nil
	}

	image := p.config.Spec.Dtr.GetBootstrapperImage()
	runFlags := []string{"--rm", "-i"}
	if dtrLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	installFlags := p.config.Spec.Dtr.InstallFlags

	if p.config.Spec.Dtr.ReplicaConfig == "sequential" {
		log.Debugf("Configuring DTR replica ids to be sequential")
		installFlags = append(installFlags, fmt.Sprintf("--replica-id %s", dtr.SequentialReplicaID(1)))
	}

	// Configure the ucpFlags from existing UcpConfig
	ucpFlags := dtr.BuildUcpFlags(p.config)
	// Conduct the install passing the --ucp-node flag for the host provided in
	// dtrLeader.
	ucpFlags = append(ucpFlags, fmt.Sprintf("--ucp-node %s", dtrLeader.Metadata.LongHostname))

	installFlags = append(installFlags, ucpFlags...)
	installCmd := dtrLeader.Configurer.DockerCommandf("run %s %s install %s", strings.Join(runFlags, " "), image, strings.Join(installFlags, " "))
	err = dtrLeader.ExecCmd(installCmd, "", true, true)
	if err != nil {
		return fmt.Errorf("%s: failed to run DTR installer: %s", dtrLeader, err.Error())
	}

	dtrMeta, err := dtr.CollectDtrFacts(dtrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing DTR details: %s", dtrLeader.Address, err)
	}
	p.config.Spec.Dtr.Metadata = dtrMeta
	return nil
}
