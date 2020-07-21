package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/dtr"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta2"
)

// InstallDtr is the phase implementation for running the actual DTR installer
// bootstrap
type InstallDtr struct {
	Analytics
}

// Title prints the phase title
func (p *InstallDtr) Title() string {
	return "Install DTR components"
}

// Run the installer container
func (p *InstallDtr) Run(config *api.ClusterConfig) (err error) {
	dtrLeader := config.Spec.DtrLeader()

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
		"dtr_version": config.Spec.Dtr.Version,
	}

	if config.Spec.Dtr.Metadata.Installed {
		log.Infof("%s: DTR already installed at version %s, not running installer", dtrLeader.Address, config.Spec.Dtr.Metadata.InstalledVersion)
		return nil
	}

	image := config.Spec.Dtr.GetBootstrapperImage()
	runFlags := []string{"--rm", "-i"}
	if dtrLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	installFlags := config.Spec.Dtr.InstallFlags

	if config.Spec.Dtr.ReplicaConfig == "sequential" {
		log.Debugf("Configuring DTR replica ids to be sequential")
		installFlags = append(installFlags, fmt.Sprintf("--replica-id %s", dtr.SequentialReplicaID(1)))
	}

	if licenseFilePath := config.Spec.Dtr.LicenseFilePath; licenseFilePath != "" {
		log.Debugf("Installing DTR with licenseFilePath: %s", licenseFilePath)
		licenseFlag, err := util.SetupLicenseFile(config.Spec.Dtr.LicenseFilePath)
		if err != nil {
			return fmt.Errorf("error while reading license file %s: %v", licenseFilePath, err)
		}
		installFlags = append(installFlags, licenseFlag)
	}

	// Configure the ucpFlags from existing UcpConfig
	ucpFlags := dtr.BuildUcpFlags(config)
	// Conduct the install passing the --ucp-node flag for the host provided in
	// dtrLeader.
	ucpFlags = append(ucpFlags, fmt.Sprintf("--ucp-node %s", dtrLeader.Metadata.Hostname))

	installFlags = append(installFlags, ucpFlags...)
	installCmd := dtrLeader.Configurer.DockerCommandf("run %s %s install %s", strings.Join(runFlags, " "), image, strings.Join(installFlags, " "))
	err = dtrLeader.ExecCmd(installCmd, "", true, true)
	if err != nil {
		return NewError("Failed to run DTR installer")
	}

	dtrMeta, err := dtr.CollectDtrFacts(dtrLeader)
	if err != nil {
		return fmt.Errorf("%s: failed to collect existing DTR details: %s", dtrLeader.Address, err)
	}
	config.Spec.Dtr.Metadata = dtrMeta
	return nil
}
