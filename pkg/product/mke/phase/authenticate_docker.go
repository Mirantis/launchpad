package phase

import (
	"fmt"
	"os"

	github.com/Mirantis/launchpad/pkg/phase"
	github.com/Mirantis/launchpad/pkg/product/mke/api"
)

// AuthenticateDocker phase implementation.
type AuthenticateDocker struct {
	phase.Analytics
	phase.BasicPhase
}

// ShouldRun is true when registry credentials are set.
func (p *AuthenticateDocker) ShouldRun() bool {
	return os.Getenv("REGISTRY_USERNAME") != "" && os.Getenv("REGISTRY_PASSWORD") != ""
}

// Title for the phase.
func (p *AuthenticateDocker) Title() string {
	return "Authenticate docker"
}

// Run authenticates docker on hosts.
func (p *AuthenticateDocker) Run() error {
	imageRepo := p.Config.Spec.MKE.ImageRepo

	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, func(h *api.Host, _ *api.ClusterConfig) error {
		if err := h.AuthenticateDocker(imageRepo); err != nil {
			return fmt.Errorf("%s: authenticate docker: %w", h, err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	return nil
}
