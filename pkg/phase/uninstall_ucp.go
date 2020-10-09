package phase

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/exec"
	"github.com/Mirantis/mcc/pkg/swarm"
	log "github.com/sirupsen/logrus"
)

// UninstallUCP is the phase implementation for running UCP uninstall
type UninstallUCP struct {
	Analytics
	BasicPhase
}

// Title prints the phase title
func (p *UninstallUCP) Title() string {
	return "Uninstall UCP components"
}

// Run the installer container
func (p *UninstallUCP) Run() error {
	swarmLeader := p.config.Spec.SwarmLeader()
	if !p.config.Spec.Ucp.Metadata.Installed {
		log.Infof("%s: UCP is not installed, skipping", swarmLeader.Address)
		return nil
	}

	image := fmt.Sprintf("%s/ucp:%s", p.config.Spec.Ucp.ImageRepo, p.config.Spec.Ucp.Version)
	args := fmt.Sprintf("--id %s", swarm.ClusterID(swarmLeader))
	runFlags := []string{"--rm", "-i", "-v /var/run/docker.sock:/var/run/docker.sock"}
	if swarmLeader.Configurer.SELinuxEnabled() {
		runFlags = append(runFlags, "--security-opt label=disable")
	}
	uninstallCmd := swarmLeader.Configurer.DockerCommandf("run %s %s uninstall-ucp %s", strings.Join(runFlags, " "), image, args)
	err := swarmLeader.Exec(uninstallCmd, exec.StreamOutput(), exec.Redact("admin-*"))
	if err != nil {
		return NewError("Failed to run UCP uninstaller")
	}

	if p.config.Spec.Ucp.CertData != "" {
		managers := p.config.Spec.Managers()
		managers.ParallelEach(func(h *api.Host) error {
			log.Infof("%s: removing ucp-controller-server-certs volume", h.Address)
			err := h.Exec(h.Configurer.DockerCommandf("volume rm --force ucp-controller-server-certs"))
			if err != nil {
				log.Errorf("%s: failed to remove the volume", h.Address)
			}
			return nil
		})
	}
	return nil
}
