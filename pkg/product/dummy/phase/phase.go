package phase

import (
	"github.com/Mirantis/mcc/pkg/product/dummy/api"
	log "github.com/sirupsen/logrus"
)

// BasicPhase is a phase which has all the basic functionality like Title and default implementations for Prepare and ShouldRun
type BasicPhase struct {
	Config *api.ClusterConfig
}

// Prepare rceives the cluster config and stores it to the phase's config field
func (p *BasicPhase) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	return nil
}

// ShouldRun for BasicPhases is always true
func (p *BasicPhase) ShouldRun() bool {
	return true
}

// CleanUp basic implementation
func (p *BasicPhase) CleanUp() {}

// RunParallelOnHosts runs a function parallelly on the listed hosts
func RunParallelOnHosts(hosts api.Hosts, config *api.ClusterConfig, action func(h *api.Host, config *api.ClusterConfig) error) error {
	return hosts.ParallelEach(func(h *api.Host) error {
		err := action(h, config)
		if err != nil {
			log.Error(err.Error())
		}
		return err
	})
}
