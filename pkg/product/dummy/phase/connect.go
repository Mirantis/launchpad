package phase

import (
	"strings"
	"time"

	"github.com/Mirantis/mcc/pkg/product/dummy/api"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// Connect connects to each of the hosts
type Connect struct {
	BasicPhase
}

// Title for the phase
func (p *Connect) Title() string {
	return "Open Remote Connection"
}

// Run connects to all the hosts in parallel
func (p *Connect) Run() error {
	return RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.connectHost)
}

const retries = 60

func (p *Connect) connectHost(h *api.Host, c *api.ClusterConfig) error {
	err := retry.Do(
		func() error {
			return h.Connect()
		},
		retry.OnRetry(
			func(n uint, err error) {
				log.Errorf("%s: attempt %d of %d.. failed to connect: %s", h, n+1, retries, err.Error())
			},
		),
		retry.RetryIf(
			func(err error) bool {
				return !strings.Contains(err.Error(), "no supported methods remain")
			},
		),
		retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
		retry.MaxJitter(time.Second*2),
		retry.Delay(time.Second*3),
		retry.Attempts(retries),
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
