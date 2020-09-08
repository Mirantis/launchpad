package phase

import (
	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	log "github.com/sirupsen/logrus"
)

// UninstallEngine phase implementation
type UninstallEngine struct {
	Analytics
	BasicPhase
}

// Title for the phase
func (p *UninstallEngine) Title() string {
	return "Uninstall Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *UninstallEngine) Run() error {
	return runParallelOnHosts(p.config.Spec.Hosts, p.config, p.uninstallEngine)
}

func (p *UninstallEngine) uninstallEngine(host *api.Host, c *api.ClusterConfig) error {
	err := host.Exec(host.Configurer.DockerCommandf("info"))
	if err != nil {
		log.Infof("%s: engine not installed, skipping", host.Address)
		return nil
	}
	log.Infof("%s: uninstalling engine", host.Address)
	err = host.Configurer.UninstallEngine(&c.Spec.Engine)
	if err == nil {
		log.Infof("%s: engine uninstalled", host.Address)
	}

	return err
}
