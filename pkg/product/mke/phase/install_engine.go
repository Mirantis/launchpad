package phase

import (
	"fmt"

	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"

	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// InstallEngine phase implementation
type InstallEngine struct {
	phase.Analytics
	phase.HostSelectPhase
}

// HostFilterFunc returns true for hosts that do not have engine installed
func (p *InstallEngine) HostFilterFunc(h *api.Host) bool {
	return h.Metadata.EngineVersion == ""
}

// Prepare collects the hosts
func (p *InstallEngine) Prepare(config *api.ClusterConfig) error {
	p.Config = config
	log.Debugf("collecting hosts for phase %s", p.Title())
	hosts := config.Spec.Hosts.Filter(p.HostFilterFunc)
	log.Debugf("found %d hosts for phase %s", len(hosts), p.Title())
	p.Hosts = hosts
	return nil
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

	return p.Hosts.ParallelEach(p.installEngine)
}

func (p *InstallEngine) installEngine(h *api.Host) error {
	err := retry.Do(
		func() error {
			log.Infof("%s: installing engine (%s)", h, p.Config.Spec.Engine.Version)
			return h.Configurer.InstallEngine(&p.Config.Spec.Engine)
		},
	)
	if err != nil {
		log.Errorf("%s: failed to install engine -> %s", h, err.Error())
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
		return fmt.Errorf("%s: engine version not %s after installation", h, p.Config.Spec.Engine.Version)
	}

	log.Infof("%s: engine version %s installed", h, p.Config.Spec.Engine.Version)
	h.Metadata.EngineVersion = p.Config.Spec.Engine.Version
	h.Metadata.EngineRestartRequired = false
	return nil
}
