package phase

import (
	"fmt"
	"math"
	"sync"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/Mirantis/mcc/pkg/phase"

	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// InstallEngine phase implementation
type InstallEngine struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase
func (p *InstallEngine) Title() string {
	return "Install Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *InstallEngine) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.Engine.Version,
	}
	err := p.upgradeEngines(p.Config)
	if err != nil {
		return err
	}

	newHosts := []*api.Host{}
	for _, h := range p.Config.Spec.Hosts {
		if h.Metadata.EngineVersion == "" {
			newHosts = append(newHosts, h)
		}
	}

	return phase.RunParallelOnHosts(newHosts, p.Config, p.installEngine)
}

// Upgrades host docker engines, first managers (one-by-one) and then ~10% rolling update to workers
// TODO: should we drain?
func (p *InstallEngine) upgradeEngines(c *api.ClusterConfig) error {
	for _, h := range c.Spec.Managers() {
		if h.Metadata.EngineVersion != "" && h.Metadata.EngineVersion != c.Spec.Engine.Version {
			err := p.installEngine(h, c)
			if err != nil {
				return err
			}
			if c.Spec.MKE.Metadata.Installed {
				err := c.Spec.CheckMKEHealthLocal(h)
				if err != nil {
					return err
				}
			}
		} else if h.Metadata.EngineVersion != "" {
			log.Infof("%s: Engine is already at version %s", h, h.Metadata.EngineVersion)
		}
	}

	workers := []*api.Host{}
	for _, h := range c.Spec.WorkersAndMSRs() {
		if h.Metadata.EngineVersion != "" && h.Metadata.EngineVersion != c.Spec.Engine.Version {
			workers = append(workers, h)
		}
	}

	// sacrifice 10% of workers for upgrade gods
	concurrentUpgrades := int(math.Floor(float64(len(workers)) * 0.10))
	if concurrentUpgrades == 0 {
		concurrentUpgrades = 1
	}
	wp := workerpool.New(concurrentUpgrades)
	mu := sync.Mutex{}
	installErrors := &phase.Error{}
	for _, w := range workers {
		if w.Metadata.EngineVersion != "" {
			h := w
			wp.Submit(func() {
				err := p.installEngine(h, c)
				if err != nil {
					mu.Lock()
					installErrors.Errors = append(installErrors.Errors)
					mu.Unlock()
				}
			})
		}
	}
	wp.StopWait()
	if installErrors.Count() > 0 {
		return installErrors
	}
	return nil
}

func (p *InstallEngine) installEngine(h *api.Host, c *api.ClusterConfig) error {
	newInstall := h.Metadata.EngineVersion == ""
	prevVersion := h.Metadata.EngineVersion

	err := retry.Do(
		func() error {
			if newInstall {
				log.Infof("%s: installing engine (%s)", h, c.Spec.Engine.Version)
			} else {
				log.Infof("%s: updating engine (%s -> %s)", h, prevVersion, c.Spec.Engine.Version)
			}
			return h.Configurer.InstallEngine(&c.Spec.Engine)
		},
	)
	if err != nil {
		if newInstall {
			log.Errorf("%s: failed to install engine -> %s", h, err.Error())
		} else {
			log.Errorf("%s: failed to update engine -> %s", h, err.Error())
		}

		return err
	}

	currentVersion, err := h.EngineVersion()
	if err != nil {
		if err := h.Reboot(); err != nil {
			return err
		}
		currentVersion, err = h.EngineVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query engine version after installation: %s", h, err.Error())
		}
	}

	if !newInstall && currentVersion == prevVersion {
		err = h.Configurer.RestartEngine()
		if err != nil {
			return fmt.Errorf("%s: failed to restart engine", h)
		}
	}

	log.Infof("%s: engine version %s installed", h, c.Spec.Engine.Version)
	return nil
}
