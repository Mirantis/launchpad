package phase

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// UpgradeEngine phase implementation
type UpgradeEngine struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that do not have engine installed
func (p *UpgradeEngine) HostFilterFunc(h *api.Host) bool {
	return h.Metadata.EngineVersion != p.Config.Spec.Engine.Version
}

// Prepare collects the hosts
func (p *UpgradeEngine) Prepare(config *api.ClusterConfig) error {
	p.Config = config
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *UpgradeEngine) Title() string {
	return "Upgrade Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *UpgradeEngine) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.Engine.Version,
	}
	return p.upgradeEngines()
}

// Upgrades host docker engines, first managers (one-by-one) and then ~10% rolling update to workers
// TODO: should we drain?
func (p *UpgradeEngine) upgradeEngines() error {
	var managers api.Hosts
	var workers api.Hosts
	var msrs api.Hosts
	for _, h := range p.Hosts {
		switch h.Role {
		case "manager":
			managers = append(managers, h)
		case "worker":
			workers = append(workers, h)
		case "msr":
			msrs = append(msrs, h)
		default:
			return fmt.Errorf("%s: unknown role: %s", h, h.Role)
		}
	}

	// Upgrade managers individually
	for _, h := range managers {
		err := p.upgradeEngine(h)
		if err != nil {
			return err
		}
		if p.Config.Spec.MKE.Metadata.Installed {
			err := p.Config.Spec.CheckMKEHealthLocal(h)
			if err != nil {
				return err
			}
		}
	}

	// Upgrade MSR hosts individually
	for _, h := range msrs {
		err := p.upgradeEngine(h)
		if err != nil {
			return err
		}
		if h.MSRMetadata.Installed {
			err := retry.Do(
				func() error {
					if _, err := msr.CollectFacts(h); err != nil {
						return err
					}
					return nil
				},
				retry.DelayType(retry.CombineDelay(retry.FixedDelay, retry.RandomDelay)),
				retry.MaxJitter(time.Second*2),
				retry.Delay(time.Second*5),
				retry.Attempts(3),
			)
			if err != nil {
				return err
			}
		}
	}

	// Upgrade worker hosts parallelly in 10% chunks
	concurrentUpgrades := int(math.Floor(float64(len(workers)) * 0.10))
	if concurrentUpgrades == 0 {
		concurrentUpgrades = 1
	}
	wp := workerpool.New(concurrentUpgrades)
	mu := sync.Mutex{}
	installErrors := &phase.Error{}
	for _, w := range workers {
		h := w
		wp.Submit(func() {
			err := p.upgradeEngine(h)
			if err != nil {
				mu.Lock()
				installErrors.Errors = append(installErrors.Errors, err)
				mu.Unlock()
			}
		})
	}
	wp.StopWait()
	if installErrors.Count() > 0 {
		return installErrors
	}
	return nil
}

func (p *UpgradeEngine) upgradeEngine(h *api.Host) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: upgrading engine (%s -> %s)", h, h.Metadata.EngineVersion, p.Config.Spec.Engine.Version)
			return h.Configurer.InstallEngine(&p.Config.Spec.Engine)
		},
	)
	if err != nil {
		log.Errorf("%s: failed to update engine -> %s", h, err.Error())
		return err
	}

	// TODO: This excercise is duplicated in InstallEngine, maybe
	// combine into some kind of "EnsureVersion"
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

	if currentVersion != p.Config.Spec.Engine.Version {
		err = h.Configurer.RestartEngine()
		if err != nil {
			return fmt.Errorf("%s: failed to restart engine", h)
		}
		currentVersion, err = h.EngineVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query engine version after restart: %s", h, err.Error())
		}
	}

	if currentVersion != p.Config.Spec.Engine.Version {
		return fmt.Errorf("%s: engine version not %s after upgrade", h, p.Config.Spec.Engine.Version)
	}

	log.Infof("%s: upgraded to engine version %s", h, p.Config.Spec.Engine.Version)
	h.Metadata.EngineVersion = p.Config.Spec.Engine.Version
	h.Metadata.EngineRestartRequired = false
	return nil
}
