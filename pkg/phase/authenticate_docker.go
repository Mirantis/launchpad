package phase

import (
	"os"

	"github.com/Mirantis/mcc/pkg/api"
)

// AuthenticateDocker phase implementation
type AuthenticateDocker struct {
	Analytics
	BasicPhase
}

// ShouldRun is true when registry credentials are set
func (p *AuthenticateDocker) ShouldRun() bool {
	return os.Getenv("REGISTRY_USERNAME") != "" && os.Getenv("REGISTRY_PASSWORD") != ""
}

// Title for the phase
func (p *AuthenticateDocker) Title() string {
	return "Authenticate docker"
}

// Run authenticates docker on hosts
func (p *AuthenticateDocker) Run() error {
	imageRepo := p.config.Spec.Ucp.ImageRepo

	return runParallelOnHosts(p.config.Spec.Hosts, p.config, func(h *api.Host, c *api.ClusterConfig) error {
		return h.AuthenticateDocker(imageRepo)
	})
}
