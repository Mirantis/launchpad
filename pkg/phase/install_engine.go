package phase

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Mirantis/mcc/pkg/analytics"
	"github.com/Mirantis/mcc/pkg/config"
	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// InstallEngine phase implementation
type InstallEngine struct{}

// Title for the phase
func (p *InstallEngine) Title() string {
	return "Install Docker EE Engine on the hosts"
}

// Run installs the engine on each host
func (p *InstallEngine) Run(c *config.ClusterConfig) error {
	start := time.Now()
	err := p.upgradeEngines(c)
	if err != nil {
		return err
	}
	newHosts := []*config.Host{}
	for _, h := range c.Hosts {
		if h.Metadata.EngineVersion == "" {
			newHosts = append(newHosts, h)
		}
	}
	err = runParallelOnHosts(newHosts, c, p.installEngine)
	if err == nil {
		duration := time.Since(start)
		props := analytics.NewAnalyticsEventProperties()
		props["duration"] = duration.Seconds()
		props["engine_version"] = c.Engine.Version
		analytics.TrackEvent("Engine Installed", props)
	}
	return err
}

// Upgrades host docker engines, first managers (one-by-one) and then ~10% rolling update to workers
// TODO: should we drain?
func (p *InstallEngine) upgradeEngines(c *config.ClusterConfig) error {
	for _, h := range c.Managers() {
		if h.Metadata.EngineVersion != "" && h.Metadata.EngineVersion != c.Engine.Version {
			err := p.installEngine(h, c)
			if err != nil {
				return err
			}
			err = retry.Do( // wait for ucp api to be healthy
				func() error {
					return h.Exec("curl -k -f https://localhost/_ping")
				},
				retry.Attempts(20),
			)
			if err != nil {
				return err
			}
		}
	}

	workers := []*config.Host{}
	for _, h := range c.Workers() {
		if h.Metadata.EngineVersion != "" && h.Metadata.EngineVersion != c.Engine.Version {
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
	installErrors := &Error{}
	for _, host := range workers {
		if host.Metadata.EngineVersion != "" {
			h := host
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

func (p *InstallEngine) installEngine(host *config.Host, c *config.ClusterConfig) error {
	newInstall := host.Metadata.EngineVersion == ""
	prevVersion := resolveEngineVersion(host)
	err := retry.Do(
		func() error {
			if newInstall {
				log.Infof("%s: installing engine (%s)", host.Address, c.Engine.Version)
			} else {
				log.Infof("%s: updating engine (%s -> %s)", host.Address, prevVersion, c.Engine.Version)
			}
			return host.Configurer.InstallEngine(&c.Engine)
		},
	)
	if err != nil {
		if newInstall {
			log.Errorf("%s: failed to install engine -> %s", host.Address, err.Error())
		} else {
			log.Errorf("%s: failed to update engine -> %s", host.Address, err.Error())
		}

		return err
	}

	currentVersion := resolveEngineVersion(host)
	if !newInstall && currentVersion == prevVersion {
		err = host.Configurer.RestartEngine()
		if err != nil {
			return NewError(fmt.Sprintf("%s: failed to restart engine", host.Address))
		}
	}

	log.Printf("%s: engine version %s installed", host.Address, c.Engine.Version)
	return nil
}
