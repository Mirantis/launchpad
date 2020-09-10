package phase

import (
	"fmt"
	"time"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/dtr"
	"github.com/Mirantis/mcc/pkg/pollutil"
	"github.com/sirupsen/logrus"
)

// ValidateUcpHealth phase implementation
type ValidateUcpHealth struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *ValidateUcpHealth) Title() string {
	return "Validating UCP Health"
}

// Run validates the health of UCP is sane before continuing with other
// launchpad phases, should be used when installing products that depend
// on UCP, such as DTR
func (p *ValidateUcpHealth) Run() error {
	// Issue a health check to the UCP san host until we receive an 'ok' status
	ucpURL := dtr.GetUcpURL(p.config)
	swarmLeader := p.config.Spec.SwarmLeader()

	pollConfig := pollutil.InfoPollfConfig("Performing health check against UCP: %s", ucpURL)
	pollConfig.NumRetries = 5
	// Poll for health every 30 seconds 5 times
	pollConfig.Interval = 30 * time.Second
	err := pollutil.Pollf(pollConfig)(func() error {
		return p.healthCheckUcp(swarmLeader, ucpURL)
	})
	if err != nil {
		return fmt.Errorf("failed to determine health of UCP: %s", err)
	}

	logrus.Info("UCP health check succeeded")
	return nil
}

func (p *ValidateUcpHealth) healthCheckUcp(host *api.Host, ucpURL string) error {
	// Use curl to check the response code of the /_ping endpoint
	output, err := host.ExecWithOutput(fmt.Sprintf(`curl -kso /dev/null -w "%%{http_code}" https://%s/_ping`, ucpURL))
	logrus.Debugf("UCP health check response code: %s, expected 200", output)
	if err != nil {
		return err
	}
	if output != "200" {
		return fmt.Errorf("unexpected response code")
	}
	return nil
}
