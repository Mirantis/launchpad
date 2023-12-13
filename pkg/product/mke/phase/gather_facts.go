package phase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/centos"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/enterpriselinux"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/oracle"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/sles"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/ubuntu"
	// needed to load the build func in package init.
	_ "github.com/Mirantis/mcc/pkg/configurer/windows"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/mke"
	"github.com/Mirantis/mcc/pkg/msr/msr2"
	"github.com/Mirantis/mcc/pkg/msr/msr3"
	"github.com/Mirantis/mcc/pkg/phase"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/Mirantis/mcc/pkg/swarm"
	"github.com/k0sproject/dig"
	log "github.com/sirupsen/logrus"
)

var (
	errCannotInstallMSR3WithMSR2 = errors.New("cannot install MSR v2 when MSR v3 is already installed, please uninstall MSR v3 first or modify the 'spec.msr.version' field in the cluster configuration to a 3.x version")
	errCannotInstallMSR2WithMSR3 = errors.New("cannot install MSR v3 when MSR v2 is already installed, if you wish to migrate to MSR v3 please use the Mirantis Migration Tool (mmt) or modify the 'spec.msr.version' field in the cluster configuration to a 2.x version")
)

// GatherFacts phase implementation to collect facts (OS, version etc.) from hosts.
type GatherFacts struct {
	phase.Analytics
	phase.BasicPhase
}

// Title for the phase.
func (p *GatherFacts) Title() string {
	return "Gather Facts"
}

// Run collect all the facts from hosts in parallel.
func (p *GatherFacts) Run() error {
	err := phase.RunParallelOnHosts(p.Config.Spec.Hosts, p.Config, p.investigateHost)
	if err != nil {
		return fmt.Errorf("failed to gather facts: %w", err)
	}
	// Gather MKE related facts
	swarmLeader := p.Config.Spec.SwarmLeader()

	// If engine is installed, we can collect some MKE & Swarm related info too
	if swarmLeader.Metadata.MCRVersion != "" {
		err := mke.CollectFacts(swarmLeader, p.Config.Spec.MKE.Metadata)
		if err != nil {
			return fmt.Errorf("%s: failed to collect existing MKE details: %w", swarmLeader, err)
		}
		if p.Config.Spec.MKE.Metadata.Installed {
			log.Infof("%s: MKE has version %s", swarmLeader, p.Config.Spec.MKE.Metadata.InstalledVersion)
		} else {
			log.Infof("%s: MKE is not installed", swarmLeader)
		}
		p.Config.Spec.MKE.Metadata.ClusterID = swarm.ClusterID(swarmLeader)
	}

	if p.Config.Spec.ContainsMSR() {
		// If we intend to configure MSR as well, gather facts for MSR
		if p.Config.Spec.MSR == nil {
			p.Config.Spec.MSR = &api.MSRConfig{}
		}

		// Collect both sets of facts to understand versions of MSR on the
		// target hosts, this is to ensure that if the user has MSR2 already
		// installed but is trying to install MSR3 we can tell them that they
		// need to use 'mmt' to migrate the versions.  The same goes for MSR3
		// to MSR2, if they have MSR3 installed and are trying to install MSR2
		// we can tell them they need to manually uninstall MSR3 first.
		// TODO: Perhaps we can call mmt for them in the future.
		msr2Installed := p.collectMSR2Facts()
		msr3Installed := p.collectMSR3Facts()

		if msr3Installed && p.Config.Spec.MSR.MajorVersion() == 2 {
			return errCannotInstallMSR2WithMSR3
		}

		if msr2Installed && p.Config.Spec.MSR.MajorVersion() == 3 {
			return errCannotInstallMSR3WithMSR2
		}
	}

	return nil
}

// collectMSR2Facts collects MSR2 facts from the hosts populating the host
// metadata struct, returning true if MSR2 is installed.
func (p *GatherFacts) collectMSR2Facts() bool {
	var installed bool

	msrHosts := p.Config.Spec.MSRs()
	err := msrHosts.ParallelEach(func(h *api.Host) error {
		if h.Metadata != nil && h.Metadata.MCRVersion != "" {
			msrMeta, err := msr2.CollectFacts(h)
			if err != nil {
				log.Debugf("%s: failed to collect existing MSR details: %s", h, err.Error())
			}
			h.MSRMetadata = msrMeta
			if msrMeta.Installed {
				log.Infof("%s: MSR has version %s", h, msrMeta.InstalledVersion)
				installed = true
			} else {
				log.Infof("%s: MSR is not installed", h)
			}
		}
		return nil
	})
	if err != nil {
		log.Debugf("failed to collect existing MSR details across MSR hosts: %s", err.Error())
	}

	return installed
}

// collectMSR3Facts collects MSR3 facts from the hosts populating the host
// metadata struct, returning true if MSR3 is installed.
func (p *GatherFacts) collectMSR3Facts() bool {
	kubeClient, helmClient, err := mke.KubeAndHelmFromConfig(p.Config)
	if err != nil {
		if errors.Is(err, mke.ErrMKENotInstalled) {
			log.Infof("mke is not yet installed, skipping msr fact collection")
			return false
		}

		log.Debugf("failed to collect existing MSR details: cannot create Helm and Kubernetes clients: %s", err.Error())
		return false
	}

	rc, err := kubeClient.GetMSRResourceClient()
	if err != nil {
		log.Debugf("failed to collect existing MSR details: cannot create MSR resource client: %s", err.Error())
		return false
	}

	msrMeta, err := msr3.CollectFacts(context.Background(), p.Config.Spec.MSR.V3.CRD.GetName(), kubeClient, rc, helmClient, kubeclient.WithCustomWait(1, time.Second*30))
	if err != nil {
		log.Debugf("failed to collect existing MSR details: %s", err.Error())
		return false
	}

	if msrMeta.Installed {
		log.Infof("MSR has version %s", msrMeta.InstalledVersion)
	} else {
		log.Info("MSR is not installed")
	}

	// Write the msrMeta to each of the hosts so it can be queried
	// independently and function like legacy MSR.
	msrHosts := p.Config.Spec.MSRs()
	err = msrHosts.ParallelEach(func(h *api.Host) error {
		h.MSRMetadata = msrMeta
		return nil
	})
	if err != nil {
		log.Debugf("failed to write MSR metadata to MSR hosts: %s", err.Error())
		return false
	}

	return msrMeta.Installed
}

var errInvalidIP = errors.New("invalid IP address")

func (p *GatherFacts) investigateHost(h *api.Host, _ *api.ClusterConfig) error {
	log.Infof("%s: gathering host facts", h)
	if h.Metadata == nil {
		h.Metadata = &api.HostMetadata{}
	}

	if err := h.Configurer.CheckPrivilege(h); err != nil {
		return fmt.Errorf("privilege check failed: %w", err)
	}

	version, err := h.MCRVersion()
	if err != nil || version == "" {
		log.Infof("%s: mirantis container runtime not installed", h)
	} else {
		log.Infof("%s: is running mirantis container runtime version %s", h, version)
		configData, err := h.Configurer.ReadFile(h, "/etc/docker/daemon.json")
		if err == nil {
			var newCfg dig.Mapping
			if err = json.Unmarshal([]byte(configData), &newCfg); err == nil {
				for k, v := range newCfg {
					if _, ok := h.DaemonConfig[k]; !ok {
						log.Debugf("%s: set %s = %t for spec.hosts[].daemonConfig from existing daemon.json", h, k, v)
						h.DaemonConfig[k] = v
					}
				}
			}
		}
	}

	h.Metadata.MCRVersion = version

	h.Metadata.Hostname = h.Configurer.Hostname(h)
	h.Metadata.LongHostname = h.Configurer.LongHostname(h)

	if h.PrivateInterface == "" {
		i, err := h.Configurer.ResolvePrivateInterface(h)
		if err != nil {
			return fmt.Errorf("%s: failed to resolve private interface: %w", h, err)
		}
		log.Infof("%s: detected private interface '%s'", h, i)
		h.PrivateInterface = i
	}

	a, err := h.Configurer.ResolveInternalIP(h, h.PrivateInterface, h.Address())
	if err != nil {
		return fmt.Errorf("%s: failed to resolve internal address: %w", h, err)
	}
	if net.ParseIP(a) == nil {
		return fmt.Errorf("%s: %w: failed to resolve internal address: invalid IP address: %q", h, errInvalidIP, a)
	}
	h.Metadata.InternalAddress = a

	log.Infof("%s: is running \"%s\"", h, h.OSVersion.String())
	log.Infof("%s: internal address: %s", h, h.Metadata.InternalAddress)

	log.Infof("%s: gathered all facts", h)

	return nil
}
