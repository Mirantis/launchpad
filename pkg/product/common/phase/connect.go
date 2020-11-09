package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *Connect) Title() string {
	return "Open Remote Connection"
}

// Run connects to all the hosts in parallel
func (p *Connect) Run() error {
	return phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.connectHost)
}

func (p *Connect) connectHost(h *api.Host, c *api.ClusterConfig) error {
	err := retry.Do(
		func() error {
			return h.Connect()
		},
		retry.Attempts(6),
	)

	if err != nil {
		return err
	}

	return p.testConnection(h)
}

func (p *Connect) testConnection(h *api.Host) error {
	log.Infof("%s: testing connection", h)

	if err := h.Exec("echo"); err != nil {
		return err
	}

	return nil
}
