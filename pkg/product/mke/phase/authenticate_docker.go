package phase

import (
	"os"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
)

// AuthenticateDocker phase implementation
type AuthenticateDocker struct {
	phase.Analytics
	phase.BasicPhase
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
	imageRepo := p.Config.Spec.MKE.ImageRepo

	return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, func(h *api.Host, c *api.ClusterConfig) error {
		return h.AuthenticateDocker(imageRepo)
	})
}
