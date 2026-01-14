package phase

import (
	"fmt"

	"github.com/Mirantis/launchpad/pkg/mke"
	"github.com/Mirantis/launchpad/pkg/phase"
	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/Mirantis/launchpad/pkg/swarm"
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
	leader := p.Config.Spec.SwarmLeader()
	if !p.Config.Spec.MKE.Metadata.Installed {
		log.Infof("%s: MKE is not installed, skipping", leader)
		return nil
	}

	uninstallFlags := commonconfig.Flags{"--id", swarm.ClusterID(leader), "--purge-config"}

	if _, err := mke.Bootstrap("uninstall-ucp", *p.Config, mke.BootstrapOptions{OperationFlags: uninstallFlags, ExecOptions: []exec.Option{exec.StreamOutput()}}); err != nil {
		return fmt.Errorf("%s: failed to run MKE uninstaller: %w", leader, err)
	}

	managers := p.Config.Spec.Managers()
	_ = managers.ParallelEach(func(h *mkeconfig.Host) error {
		log.Infof("%s: removing ucp-controller-server-certs volume", h)
		err := h.Exec(h.Configurer.DockerCommandf("volume rm --force ucp-controller-server-certs"))
		if err != nil {
			log.Errorf("%s: failed to remove the volume", h)
		}

		if err := h.Reboot(); err != nil {
			log.Errorf("%s: failed to reboot the host: %v", h, err)
		}
		return nil
	})

	workers := p.Config.Spec.WorkersAndMSRs()
	_ = workers.ParallelEach(func(h *mkeconfig.Host) error {
		if err := h.Reboot(); err != nil {
			log.Errorf("%s: failed to reboot the host: %v", h, err)
		}
		return nil
	})

	return nil
}
