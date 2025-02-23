package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mirantis/launchpad/pkg/constant"
	common "github.com/Mirantis/launchpad/pkg/product/common/api"
	retry "github.com/avast/retry-go"
	"github.com/creasty/defaults"
	"github.com/k0sproject/rig"
	log "github.com/sirupsen/logrus"
)

// Cluster is for universal cluster settings not applicable to single hosts, mke, msr or engine.
type Cluster struct {
	Prune bool `yaml:"prune" default:"false"`
}

// ClusterSpec defines cluster spec.
type ClusterSpec struct {
	Hosts   Hosts            `yaml:"hosts" validate:"required,min=1,dive"`
	MKE     MKEConfig        `yaml:"mke,omitempty"`
	MSR     *MSRConfig       `yaml:"msr,omitempty"`
	MCR     common.MCRConfig `yaml:"mcr,omitempty"`
	Cluster Cluster          `yaml:"cluster"`
}

// Workers filters only the workers from the cluster config.
func (c *ClusterSpec) Workers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "worker" })
}

// Managers filters only the manager nodes from the cluster config.
func (c *ClusterSpec) Managers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "manager" })
}

// MSRs filters only the MSR nodes from the cluster config.
func (c *ClusterSpec) MSRs() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "msr" })
}

// WorkersAndMSRs filters both worker and MSR roles from the cluster config.
func (c *ClusterSpec) WorkersAndMSRs() Hosts {
	return c.Hosts.Filter(func(h *Host) bool {
		return h.Role == "msr" || h.Role == "worker"
	})
}

// SwarmLeader resolves the current swarm leader host.
func (c *ClusterSpec) SwarmLeader() *Host {
	m := c.Managers()
	leader := m.Find(isSwarmLeader)
	if leader != nil {
		log.Debugf("%s: is the swarm leader", leader)
		return leader
	}

	log.Debugf("did not find a real swarm manager, fallback to first manager host")
	return m.First()
}

var errGenerateURL = errors.New("unable to generate url")

// MKEURL returns a URL for MKE or an error if one can not be generated.
func (c *ClusterSpec) MKEURL() (*url.URL, error) {
	// Easy route, user has provided one in MSR --ucp-url
	if c.MSR != nil {
		if f := c.MSR.InstallFlags.GetValue("--ucp-url"); f != "" {
			if !strings.Contains(f, "://") {
				f = "https://" + f
			}
			u, err := url.Parse(f)
			if err != nil {
				return nil, fmt.Errorf("invalid MSR --ucp-url install flag '%s': %w", f, err)
			}
			if u.Path == "" {
				u.Path = "/"
			}
			return u, nil
		}
	}

	var mkeAddr string
	// Option 2: there's a "--san" install flag
	if addr := c.MKE.InstallFlags.GetValue("--san"); addr != "" {
		mkeAddr = addr
	} else {
		// Option 3: Use the first manager's address
		mgrs := c.Managers()
		if len(mgrs) < 1 {
			return nil, fmt.Errorf("%w: mke managers count is zero", errGenerateURL)
		}
		mkeAddr = mgrs[0].Address()
	}

	if portstr := c.MKE.InstallFlags.GetValue("--controller-port"); portstr != "" {
		p, err := strconv.Atoi(portstr)
		if err != nil {
			return nil, fmt.Errorf("invalid mke controller-port value: '%s': %w", portstr, err)
		}
		mkeAddr = fmt.Sprintf("%s:%d", mkeAddr, p)
	}

	return &url.URL{
		Scheme: "https",
		Path:   "/",
		Host:   mkeAddr,
	}, nil
}

// MSRURL returns an url to MSR or an error if one can't be generated.
func (c *ClusterSpec) MSRURL() (*url.URL, error) {
	if c.MSR != nil {
		// Default to using the --dtr-external-url if it's set
		if f := c.MSR.InstallFlags.GetValue("--dtr-external-url"); f != "" {
			if !strings.Contains(f, "://") {
				f = "https://" + f
			}
			u, err := url.Parse(f)
			if err != nil {
				return nil, fmt.Errorf("invalid MSR --dtr-external-url install flag '%s': %w", f, err)
			}
			if u.Scheme == "" {
				u.Scheme = "https"
			}
			if u.Path == "" {
				u.Path = "/"
			}
			return u, nil
		}
	}

	var msrAddr string

	// Otherwise, use MSRLeaderAddress
	msrLeader := c.MSRLeader()
	if msrLeader == nil {
		return nil, fmt.Errorf("%w: no MSR nodes found", errGenerateURL)
	}
	msrAddr = msrLeader.Address()

	if c.MSR != nil {
		if portstr := c.MSR.InstallFlags.GetValue("--replica-https-port"); portstr != "" {
			p, err := strconv.Atoi(portstr)
			if err != nil {
				return nil, fmt.Errorf("invalid msr --replica-https-port value '%s': %w", portstr, err)
			}
			msrAddr = fmt.Sprintf("%s:%d", msrAddr, p)
		}
	}

	return &url.URL{
		Scheme: "https",
		Path:   "/",
		Host:   msrAddr,
	}, nil
}

var errInvalidConfig = errors.New("invalid configuration")

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml.
func (c *ClusterSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type spec ClusterSpec
	specAlias := (*spec)(c)
	c.MCR = common.MCRConfig{}
	c.MKE = NewMKEConfig()

	if err := unmarshal(specAlias); err != nil {
		return err
	}

	if c.Hosts.Count(func(h *Host) bool { return h.Role == "msr" }) > 0 {
		if specAlias.MSR == nil {
			return fmt.Errorf("%w: hosts with msr role present, but no spec.msr defined", errInvalidConfig)
		}
		if err := defaults.Set(specAlias.MSR); err != nil {
			return fmt.Errorf("set defaults: %w", err)
		}
	} else if specAlias.MSR != nil {
		specAlias.MSR = nil
		log.Debugf("ignoring spec.msr configuration as there are no hosts having the msr role")
	}

	bastionHosts := c.Hosts.Filter(func(h *Host) bool {
		return (h.SSH != nil && h.SSH.Bastion != nil) || (h.WinRM != nil && h.WinRM.Bastion != nil)
	})
	if len(bastionHosts) > 0 {
		log.Debugf("linking bastion hosts")
		bastions := make(map[string]*rig.SSH)
		for _, h := range bastionHosts {
			if h.WinRM != nil {
				id := fmt.Sprintf("%s@%s:%d", h.WinRM.User, h.WinRM.Address, h.WinRM.Port)
				bastions[id] = h.WinRM.Bastion
			} else if h.SSH != nil {
				id := fmt.Sprintf("%s@%s:%d", h.SSH.User, h.SSH.Address, h.SSH.Port)
				bastions[id] = h.SSH.Bastion
			}
		}
		for _, h := range bastionHosts {
			if h.WinRM != nil {
				id := fmt.Sprintf("%s@%s:%d", h.WinRM.User, h.WinRM.Address, h.WinRM.Port)
				h.WinRM.Bastion = bastions[id]
			} else if h.SSH != nil {
				id := fmt.Sprintf("%s@%s:%d", h.SSH.User, h.SSH.Address, h.SSH.Port)
				h.SSH.Bastion = bastions[id]
			}
		}
	}

	if err := defaults.Set(c); err != nil {
		return fmt.Errorf("set defaults: %w", err)
	}
	return nil
}

func isSwarmLeader(h *Host) bool {
	// We can by-pass the Configurer interface as managers are always linux boxes
	output, err := h.ExecOutput(h.Configurer.DockerCommandf(`info --format "{{ .Swarm.ControlAvailable}}"`))
	if err != nil {
		log.Debugf("%s: failed to get host's swarm leader status, probably not part of swarm", h)
		return false
	}
	return output == "true"
}

// IsMSRInstalled checks to see if MSR is installed on the given host.
func IsMSRInstalled(h *Host) bool {
	return h.MSRMetadata != nil && h.MSRMetadata.Installed
}

// MSRLeader returns the current MSRLeader host.
func (c *ClusterSpec) MSRLeader() *Host {
	// MSR doesn't have the concept of leaders during the installation phase,
	// but we need to make sure we have a Host to reference during our other
	// bootstrap operations: Upgrade and Join
	msrs := c.MSRs()
	h := msrs.Find(IsMSRInstalled)
	if h != nil {
		log.Debugf("%s: found MSR installed, using as leader", h)
		return h
	}

	log.Debugf("did not find a MSR installation, falling back to the first MSR host")
	return msrs.First()
}

// IsCustomImageRepo checks if the config is using a custom image repo.
func IsCustomImageRepo(imageRepo string) bool {
	return imageRepo != constant.ImageRepo && imageRepo != constant.ImageRepoLegacy
}

func pingHost(h *Host, address string, waitgroup *sync.WaitGroup, errCh chan<- error) {
	url := fmt.Sprintf("https://%s/_ping", address)

	err := retry.Do(
		func() error {
			log.Infof("%s: waiting for MKE at %s to become healthy", h, url)
			if err := h.CheckHTTPStatus(url, http.StatusOK); err != nil {
				return fmt.Errorf("check http status: %w", err)
			}
			return nil
		},
		retry.MaxJitter(time.Second*3),
		retry.Delay(time.Second*30),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(10), // should try for ~5min
	)
	if err != nil {
		errCh <- fmt.Errorf("MKE health check failed: %w", err)
	}
	errCh <- nil
	waitgroup.Done()
}

// CheckMKEHealthRemote will check mke cluster health from a list of hosts and return an error if it failed.
func (c *ClusterSpec) CheckMKEHealthRemote(hosts []*Host) error {
	errCh := make(chan error, len(hosts))
	var wg sync.WaitGroup

	for _, h := range hosts {
		wg.Add(1)
		go pingHost(h, h.Address(), &wg, errCh)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return fmt.Errorf("MKE health check failed: %w", err)
		}
	}

	return nil
}

// CheckMKEHealthLocal will check the local mke health on a host and return an error if it failed.
func (c *ClusterSpec) CheckMKEHealthLocal(hosts []*Host) error {
	errCh := make(chan error, len(hosts))
	var wg sync.WaitGroup

	for _, h := range hosts {
		wg.Add(1)
		address := h.Metadata.InternalAddress
		if port := c.MKE.InstallFlags.GetValue("--controller-port"); port != "" {
			address = address + ":" + port
		}
		go pingHost(h, address, &wg, errCh)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return fmt.Errorf("MKE health check failed: %w", err)
		}
	}

	return nil
}

// ContainsMSR returns true when the config has msr hosts.
func (c *ClusterSpec) ContainsMSR() bool {
	return c.Hosts.Find(func(h *Host) bool { return h.Role == "msr" }) != nil
}
