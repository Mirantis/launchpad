package phase

import (
	"github.com/Mirantis/mcc/pkg/api"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *Connect) Title() string {
	return "Open Remote Connection"
}

// Run connects to all the hosts in parallel
func (p *Connect) Run() error {
	return runParallelOnHosts(p.config.Spec.Hosts, p.config, p.connectHost)
}

func (p *Connect) connectHost(h *api.Host, c *api.ClusterConfig) error {
	proto := "SSH"

	if h.Localhost {
		proto = "Local"
	} else if h.WinRM != nil {
		proto = "WinRM"
	}

	err := retry.Do(
		func() error {
			log.Infof("%s: opening %s connection", h.Address, proto)
			err := h.Connect()
			if err != nil {
				log.Errorf("%s: failed to connect -> %s", h.Address, err.Error())
			}
			return err
		},
		retry.Attempts(6),
	)
	if err != nil {
		log.Errorf("%s: failed to open connection", h.Address)
		return err
	}

	log.Printf("%s: %s connection opened", h.Address, proto)
	return p.testConnection(h)
}

func (p *Connect) testConnection(h *api.Host) error {
	log.Infof("%s: testing connection", h.Address)

	if err := h.Exec("echo"); err != nil {
		return err
	}

	return nil
}
