package phase

import (
	"fmt"

	mcclog "github.com/Mirantis/mcc/pkg/log"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/phase"
	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/k0sproject/rig/exec"
	log "github.com/sirupsen/logrus"
)

// UninstallMKE is the phase implementation for running MKE uninstall.
type UninstallMKE struct {
	phase.Analytics
	phase.BasicPhase
}

// Title prints the phase title.
func (p *UninstallMKE) Title() string {
	return "Uninstall MKE components"
}

// Run the installer container.
func (p *UninstallMKE) Run() error {
	swarmLeader := p.Config.Spec.SwarmLeader()
	if !p.Config.Spec.MKE.Metadata.Installed {
		log.Infof("%s: MKE is not installed, skipping", swarmLeader)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", p.Config.Spec.MKE.ImageRepo, p.Config.Spec.MKE.Version)
	uninstallFlags := common.Flags{"--id", swarm.ClusterID(swarmLeader), "--purge-config"}

	if mcclog.Debug {
		uninstallFlags.AddUnlessExist("--debug")
	}

	runFlags := common.Flags{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled(swarmLeader) {
		runFlags.Add("--security-opt label=disable")
	}
	uninstallCmd := swarmLeader.Configurer.DockerCommandf("run %s %s uninstall-ucp %s", runFlags.Join(), image, uninstallFlags.Join())
	err := swarmLeader.Exec(uninstallCmd, exec.StreamOutput(), exec.RedactString(p.Config.Spec.MKE.InstallFlags.GetValue("--admin-username"), p.Config.Spec.MKE.InstallFlags.GetValue("--admin-password")))
	if err != nil {
		return fmt.Errorf("%s: failed to run MKE uninstaller: %w", swarmLeader, err)
	}

	if p.Config.Spec.MKE.CertData != "" {
		managers := p.Config.Spec.Managers()
		err = managers.ParallelEach(func(h *api.Host) error {
			log.Infof("%s: removing ucp-controller-server-certs volume", h)
			err := h.Exec(h.Configurer.DockerCommandf("volume rm --force ucp-controller-server-certs"))
			if err != nil {
				log.Errorf("%s: failed to remove the volume", h)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to execute volume removal across manager hosts: %w", err)
		}
	}

	mke.CleanBundleDir(p.Config)

	return nil
}
