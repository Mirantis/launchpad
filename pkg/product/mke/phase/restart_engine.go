package phase

import (
	"math"
	"sync"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// RestartEngine phase implementation
type RestartEngine struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that need their engine to be restarted
func (p *RestartEngine) HostFilterFunc(h *api.Host) bool {
	return h.Metadata.EngineRestartRequired
}

// Prepare collects the hosts
func (p *RestartEngine) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *RestartEngine) Title() string {
	return "Restart Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *RestartEngine) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.Engine.Version,
	}
	return p.restartEngines()
}

// Restarts host docker engines, first managers (one-by-one) and then ~10% rolling update to workers
func (p *RestartEngine) restartEngines() error {
	var managers api.Hosts
	var others api.Hosts
	for _, h := range p.Hosts {
		if h.Role == "manager" {
			managers = append(managers, h)
		} else {
			others = append(others, h)
		}
	}

	for _, h := range managers {
		if err := h.Configurer.RestartEngine(); err != nil {
			return err
		}
	}

	// restart in 10% chunks
	concurrentRestarts := int(math.Floor(float64(len(others)) * 0.10))
	if concurrentRestarts == 0 {
		concurrentRestarts = 1
	}
	wp := workerpool.New(concurrentRestarts)
	mu := sync.Mutex{}
	restartErrors := &phase.Error{}
	for _, w := range others {
		h := w
		wp.Submit(func() {
			err := h.Configurer.RestartEngine()
			if err != nil {
				mu.Lock()
				restartErrors.Errors = append(restartErrors.Errors, err)
				mu.Unlock()
			}
			h.Metadata.EngineRestartRequired = false
		})
	}
	wp.StopWait()
	if restartErrors.Count() > 0 {
		return restartErrors
	}
	return nil
}
