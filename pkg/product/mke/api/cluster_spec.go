package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	retry "github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
)

// Cluster is for universal cluster settings not applicable to single hosts, mke, msr or engine
type Cluster struct {
	Prune bool `yaml:"prune" default:"false"`
}

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts   Hosts        `yaml:"hosts" validate:"gt=1"`
	MKE     MKEConfig    `yaml:"mke,omitempty"`
	MSR     *MSRConfig   `yaml:"msr,omitempty"`
	Engine  EngineConfig `yaml:"engine,omitempty"`
	Cluster Cluster      `yaml:"cluster"`
}

// Workers filters only the workers from the cluster config
func (c *ClusterSpec) Workers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "worker" })
}

// Managers filters only the manager nodes from the cluster config
func (c *ClusterSpec) Managers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "manager" })
}

// MSRs filters only the MSR nodes from the cluster config
func (c *ClusterSpec) MSRs() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "msr" })
}

// WorkersAndMSRs filters both worker and MSR roles from the cluster config
func (c *ClusterSpec) WorkersAndMSRs() Hosts {
	return c.Hosts.Filter(func(h *Host) bool {
		return h.Role == "msr" || h.Role == "worker"
	})
}

// SwarmLeader resolves the current swarm leader host
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

// MKEURL returns a URL for MKE or an error if one can not be generated
func (c *ClusterSpec) MKEURL() (*url.URL, error) {
	// Easy route, user has provided one in MSR --ucp-url
	if c.MSR != nil {
		if f := c.MSR.InstallFlags.GetValue("--ucp-url"); f != "" {
			if !strings.Contains(f, "://") {
				f = "https://" + f
			}
			u, err := url.Parse(f)
			if err != nil {
				return nil, fmt.Errorf("invalid MSR --ucp-url install flag '%s': %s", f, err.Error())
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
			return nil, fmt.Errorf("unable to generate a url for mke")
		}
		mkeAddr = mgrs[0].Address
	}

	if portstr := c.MKE.InstallFlags.GetValue("--controller-port"); portstr != "" {
		p, err := strconv.Atoi(portstr)
		if err != nil {
			return nil, fmt.Errorf("invalid mke controller-port value: '%s': %s", portstr, err.Error())
		}
		mkeAddr = fmt.Sprintf("%s:%d", mkeAddr, p)
	}

	return &url.URL{
		Scheme: "https",
		Path:   "/",
		Host:   mkeAddr,
	}, nil
}

// MSRURL returns an url to MSR or an error if one can't be generated
func (c *ClusterSpec) MSRURL() (*url.URL, error) {
	if c.MSR != nil {
		// Default to using the --dtr-external-url if it's set
		if f := c.MSR.InstallFlags.GetValue("--dtr-external-url"); f != "" {
			if !strings.Contains(f, "://") {
				f = "https://" + f
			}
			u, err := url.Parse(f)
			if err != nil {
				return nil, fmt.Errorf("invalid MSR --dtr-external-url install flag '%s': %s", f, err.Error())
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
		return nil, fmt.Errorf("unable to generate a MSR URL - no MSR nodes found")
	}
	msrAddr = msrLeader.Address

	if c.MSR != nil {
		if portstr := c.MSR.InstallFlags.GetValue("--replica-https-port"); portstr != "" {
			p, err := strconv.Atoi(portstr)
			if err != nil {
				return nil, fmt.Errorf("invalid msr --replica-https-port value '%s': %s", portstr, err.Error())
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

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type spec ClusterSpec
	yc := (*spec)(c)
	c.Engine = EngineConfig{}
	c.MKE = NewMKEConfig()

	if err := unmarshal(yc); err != nil {
		return err
	}

	c.Engine.SetDefaults()

	return nil
}

func isSwarmLeader(h *Host) bool {
	// We can by-pass the Configurer interface as managers are always linux boxes
	output, err := h.ExecWithOutput(`sudo docker info --format "{{ .Swarm.ControlAvailable}}"`)
	if err != nil {
		log.Debugf("%s: failed to get host's swarm leader status, probably not part of swarm", h)
		return false
	}
	return output == "true"
}

// IsMSRInstalled checks to see if MSR is installed on the given host
func IsMSRInstalled(h *Host) bool {
	return h.MSRMetadata != nil && h.MSRMetadata.Installed
}

// MSRLeader returns the current MSRLeader host
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

// IsCustomImageRepo checks if the config is using a custom image repo
func IsCustomImageRepo(imageRepo string) bool {
	return imageRepo != constant.ImageRepo && imageRepo != constant.ImageRepoLegacy
}

// CheckMKEHealthRemote will check mke cluster health from a host and return an error if it failed
func (c *ClusterSpec) CheckMKEHealthRemote(h *Host) error {
	u, err := c.MKEURL()
	if err != nil {
		return err
	}
	u.Path = "/_ping"

	return retry.Do(
		func() error {
			log.Infof("%s: waiting for MKE at %s to become healthy", h, u.Host)
			return h.CheckHTTPStatus(u.String(), 200)
		},
		retry.Attempts(12), // last attempt should wait ~7min
	)
}

// CheckMKEHealthLocal will check the local mke health on a host and return an error if it failed
func (c *ClusterSpec) CheckMKEHealthLocal(h *Host) error {
	host := "localhost"
	if port := c.MKE.InstallFlags.GetValue("--controller-port"); port != "" {
		host = host + ":" + port
	}

	return retry.Do(
		func() error {
			log.Infof("%s: waiting for MKE to become healthy", h)
			return h.CheckHTTPStatus(fmt.Sprintf("https://%s/_ping", host), 200)
		},
		retry.Attempts(12), // last attempt should wait ~7min
	)
}

// ContainsMSR returns true when the config has msr hosts
func (c *ClusterSpec) ContainsMSR() bool {
	return c.Hosts.Find(func(h *Host) bool { return h.Role == "msr" }) != nil
}
