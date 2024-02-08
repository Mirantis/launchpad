package phase

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Mirantis/mcc/pkg/msr"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	retry "github.com/avast/retry-go"
	"github.com/gammazero/workerpool"
	log "github.com/sirupsen/logrus"
)

// UpgradeMCR phase implementation
type UpgradeMCR struct {
	phase.Analytics
	phase.HostSelectPhase

	Concurrency  int
	ForceUpgrade bool
}

// HostFilterFunc returns true for hosts that do not have engine installed
func (p *UpgradeMCR) HostFilterFunc(h *api.Host) bool {
	if h.Metadata.MCRVersion != p.Config.Spec.MCR.Version {
		return true
	}
	if p.ForceUpgrade {
		log.Warnf("%s: MCR version is already %s but attempting an upgrade anyway because --force-upgrade was given", h, h.Metadata.MCRVersion)
		return true
	}
	return false
}

// Prepare collects the hosts
func (p *UpgradeMCR) Prepare(config interface{}) error {
	p.Config = config.(*api.ClusterConfig)
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := p.Config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
}

// Title for the phase
func (p *UpgradeMCR) Title() string {
	return "Upgrade Mirantis Container Runtime on the hosts"
}

// Run installs the engine on each host
func (p *UpgradeMCR) Run() error {
	p.EventProperties = map[string]interface{}{
		"engine_version": p.Config.Spec.MCR.Version,
	}
	return p.upgradeMCRs()
}

// Upgrades host docker engines, first managers (one-by-one) and then ~10% rolling update to workers
// TODO: should we drain?
func (p *UpgradeMCR) upgradeMCRs() error {
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
		err := p.upgradeMCR(h)
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

	port := 443
	if p.Config.Spec.MSR != nil {
		if flagport := p.Config.Spec.MSR.InstallFlags.GetValue("--replica-https-port"); flagport != "" {
			if fp, err := strconv.Atoi(flagport); err == nil {
				port = fp
			}
		}
	}

	// Upgrade MSR hosts individually
	for _, h := range msrs {
		if h.MSRMetadata.Installed {
			if err := msr.WaitMSRNodeReady(h, port); err != nil {
				return err
			}
		}
		if err := p.upgradeMCR(h); err != nil {
			return err
		}
		if h.MSRMetadata.Installed {
			if err := msr.WaitMSRNodeReady(h, port); err != nil {
				return err
			}
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

	log.Debugf("concurrently upgrading workers in batches of %d", p.Concurrency)
	wp := workerpool.New(p.Concurrency)
	mu := sync.Mutex{}
	installErrors := &phase.Error{}
	for _, w := range workers {
		h := w
		wp.Submit(func() {
			err := p.upgradeMCR(h)
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

func (p *UpgradeMCR) upgradeMCR(h *api.Host) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: upgrading container runtime (%s -> %s)", h, h.Metadata.MCRVersion, p.Config.Spec.MCR.Version)
			return h.Configurer.InstallMCR(h, h.Metadata.MCRInstallScript, p.Config.Spec.MCR)
		},
	)
	if err != nil {
		log.Errorf("%s: failed to update container runtime -> %s", h, err.Error())
		return err
	}

	// TODO: This excercise is duplicated in InstallMCR, maybe
	// combine into some kind of "EnsureVersion"
	currentVersion, err := h.MCRVersion()
	if err != nil {
		if err := h.Reboot(); err != nil {
			return err
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after installation: %s", h, err.Error())
		}
	}

	if currentVersion != p.Config.Spec.MCR.Version {
		err = h.Configurer.RestartMCR(h)
		if err != nil {
			return fmt.Errorf("%s: failed to restart container runtime", h)
		}
		currentVersion, err = h.MCRVersion()
		if err != nil {
			return fmt.Errorf("%s: failed to query container runtime version after restart: %s", h, err.Error())
		}
	}

	if currentVersion != p.Config.Spec.MCR.Version {
		return fmt.Errorf("%s: container runtime version not %s after upgrade", h, p.Config.Spec.MCR.Version)
	}

	log.Infof("%s: upgraded to mirantis container runtime version %s", h, p.Config.Spec.MCR.Version)
	h.Metadata.MCRVersion = p.Config.Spec.MCR.Version
	h.Metadata.MCRRestartRequired = false
	return nil
}
