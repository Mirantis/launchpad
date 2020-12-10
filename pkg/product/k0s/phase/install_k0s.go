package phase

import (
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/k0s/api"
	log "github.com/sirupsen/logrus"
)

//InstallK0s install phase
type InstallK0s struct {
	phase.Analytics
	BasicPhase
}

// Title for the phase
func (p *InstallK0s) Title() string {
	return "Install K0s"
}

//Run executes installK0s phase
func (p *InstallK0s) Run() error {
	err := RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.installK0s)
	if err != nil {
		return err
	}

	return nil
}

func (p *InstallK0s) installK0s(h *api.Host, c *api.ClusterConfig) error {
	log.Infof("%s: getting K0s binary", h)

	return h.Configurer.InstallK0s(p.Config.Spec.K0s.Version, &p.Config.Spec.K0s.Config)
}
