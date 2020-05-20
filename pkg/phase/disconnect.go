package phase

import (
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	log "github.com/sirupsen/logrus"
)

// Disconnect phase implementation
type Disconnect struct{}

// Title for the phase
func (p *Disconnect) Title() string {
	return "Close SSH Connection"
}

// Run disconnects from all the hosts
func (p *Disconnect) Run(config *config.ClusterConfig) error {
	start := time.Now()
	err := runParallelOnHosts(config.Hosts, config, p.disconnectHost)
	if err == nil {
		duration := time.Since(start)
		props := analytics.NewAnalyticsEventProperties()
		props["duration"] = duration.Seconds()
		analytics.TrackEvent("Hosts Disconnected", props)
	}
	return err
}

func (p *Disconnect) disconnectHost(host *config.Host, c *config.ClusterConfig) error {
	host.Connect()
	log.Printf("%s: SSH connection closed", host.Address)
	return nil
}
