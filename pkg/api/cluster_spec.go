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

// ClusterSpecMetadata contains spec level metadata
type ClusterSpecMetadata struct {
	Force bool
}

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts  Hosts        `yaml:"hosts" validate:"required,dive,min=1"`
	Ucp    UcpConfig    `yaml:"ucp,omitempty"`
	Dtr    *DtrConfig   `yaml:"dtr,omitempty"`
	Engine EngineConfig `yaml:"engine,omitempty"`

	Metadata ClusterSpecMetadata `yaml:"-"`
}

// Workers filters only the workers from the cluster config
func (c *ClusterSpec) Workers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "worker" })
}

// Managers filters only the manager nodes from the cluster config
func (c *ClusterSpec) Managers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "manager" })
}

// Dtrs filters only the DTR nodes from the cluster config
func (c *ClusterSpec) Dtrs() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "dtr" })
}

// WorkersAndDtrs filters both worker and DTR roles from the cluster config
func (c *ClusterSpec) WorkersAndDtrs() Hosts {
	return c.Hosts.Filter(func(h *Host) bool {
		return h.Role == "dtr" || h.Role == "worker"
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

// UcpURL returns a URL for UCP or an error if one can not be generated
func (c *ClusterSpec) UcpURL() (*url.URL, error) {
	// Easy route, user has provided one in DTR --ucp-url
	if c.Dtr != nil {
		if f := c.Dtr.InstallFlags.GetValue("--ucp-url"); f != "" {
			if !strings.Contains(f, "://") {
				f = "https://" + f
			}
			u, err := url.Parse(f)
			if err != nil {
				return nil, fmt.Errorf("invalid DTR --ucp-url install flag '%s': %s", f, err.Error())
			}
			if u.Path == "" {
				u.Path = "/"
			}
			return u, nil
		}
	}

	var ucpAddr string
	// Option 2: there's a "--san" install flag
	if addr := c.Ucp.InstallFlags.GetValue("--san"); addr != "" {
		ucpAddr = addr
	} else {
		// Option 3: Use the first manager's address
		mgrs := c.Managers()
		if len(mgrs) < 1 {
			return nil, fmt.Errorf("unable to generate a url for ucp")
		}
		ucpAddr = mgrs[0].Address
	}

	if portstr := c.Ucp.InstallFlags.GetValue("--controller-port"); portstr != "" {
		p, err := strconv.Atoi(portstr)
		if err != nil {
			return nil, fmt.Errorf("invalid ucp controller-port value: '%s': %s", portstr, err.Error())
		}
		ucpAddr = fmt.Sprintf("%s:%d", ucpAddr, p)
	}

	return &url.URL{
		Scheme: "https",
		Path:   "/",
		Host:   ucpAddr,
	}, nil
}

// DtrURL returns an url to DTR or an error if one can't be generated
func (c *ClusterSpec) DtrURL() (*url.URL, error) {
	if c.Dtr != nil {
		// Default to using the --dtr-external-url if it's set
		if f := c.Dtr.InstallFlags.GetValue("--dtr-external-url"); f != "" {
			if !strings.Contains(f, "://") {
				f = "https://" + f
			}
			u, err := url.Parse(f)
			if err != nil {
				return nil, fmt.Errorf("invalid DTR --dtr-external-url install flag '%s': %s", f, err.Error())
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

	var dtrAddr string

	// Otherwise, use DtrLeaderAddress
	if c.Dtr != nil && c.Dtr.Metadata != nil && c.Dtr.Metadata.DtrLeaderAddress != "" {
		dtrAddr = c.Dtr.Metadata.DtrLeaderAddress
	} else {
		dtrs := c.Dtrs()
		if len(dtrs) < 1 {
			return nil, fmt.Errorf("unable to generate a DTR URL - no nodes with role 'dtr' present")
		}
		dtrAddr = dtrs[0].Address
	}

	if c.Dtr != nil {
		if portstr := c.Dtr.InstallFlags.GetValue("--replica-https-port"); portstr != "" {
			p, err := strconv.Atoi(portstr)
			if err != nil {
				return nil, fmt.Errorf("invalid dtr --replica-https-port value '%s': %s", portstr, err.Error())
			}
			dtrAddr = fmt.Sprintf("%s:%d", dtrAddr, p)
		}
	}

	return &url.URL{
		Scheme: "https",
		Path:   "/",
		Host:   dtrAddr,
	}, nil
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type spec ClusterSpec
	yc := (*spec)(c)
	c.Metadata = ClusterSpecMetadata{}
	c.Engine = EngineConfig{}
	c.Ucp = NewUcpConfig()

	if err := unmarshal(yc); err != nil {
		return err
	}

	c.Engine.SetDefaults()

	return nil
}

func isSwarmLeader(host *Host) bool {
	// We can by-pass the Configurer interface as managers are always linux boxes
	output, err := host.ExecWithOutput(`sudo docker info --format "{{ .Swarm.ControlAvailable}}"`)
	if err != nil {
		log.Debugf("%s: failed to get host's swarm leader status, probably not part of swarm", host)
		return false
	}
	return output == "true"
}

// IsDtrInstalled checks to see if DTR is installed on the given host
func IsDtrInstalled(host *Host) bool {
	output, err := host.ExecWithOutput(`sudo docker ps -q --filter name=dtr-`)
	if err != nil {
		// During the initial pre-installation phases, we expect this to fail
		// so logging the error to debug is best to prevent erroneous errors
		// from appearing problematic
		log.Debugf("unable to determine if host has DTR installed: %s", err)
		return false
	}
	output = strings.Trim(output, "\n")
	if len(output) >= 9 {
		// Check for the presence of the 9 DTR containers we expect to be
		// running
		return true
	}
	return false
}

// DtrLeader returns the current DtrLeader host
func (c *ClusterSpec) DtrLeader() *Host {
	// DTR doesn't have the concept of leaders during the installation phase,
	// but we need to make sure we have a Host to reference during our other
	// bootstrap operations: Upgrade and Join
	dtrs := c.Dtrs()
	leader := dtrs.Find(IsDtrInstalled)
	if leader != nil {
		log.Debugf("%s: found DTR installed, using as leader", leader)
		return leader
	}

	log.Debugf("did not find a DTR installation, falling back to the first DTR host")
	return dtrs.First()
}

// IsCustomImageRepo checks if the config is using a custom image repo
func IsCustomImageRepo(imageRepo string) bool {
	return imageRepo != constant.ImageRepo && imageRepo != constant.ImageRepoLegacy
}

// CheckUCPHealthRemote will check ucp cluster health from a host and return an error if it failed
func (c *ClusterSpec) CheckUCPHealthRemote(h *Host) error {
	u, err := c.UcpURL()
	if err != nil {
		return err
	}
	u.Path = "/_ping"

	return retry.Do(
		func() error {
			log.Infof("%s: waiting for UCP at %s to become healthy", h, u.Host)
			return h.CheckHTTPStatus(u.String(), 200)
		},
		retry.Attempts(12), // last attempt should wait ~7min
	)
}

// CheckUCPHealthLocal will check the local ucp health on a host and return an error if it failed
func (c *ClusterSpec) CheckUCPHealthLocal(h *Host) error {
	host := "localhost"
	if port := c.Ucp.InstallFlags.GetValue("--controller-port"); port != "" {
		host = host + ":" + port
	}

	return retry.Do(
		func() error {
			log.Infof("%s: waiting for UCP to become healthy", h)
			return h.CheckHTTPStatus(fmt.Sprintf("https://%s/_ping", host), 200)
		},
		retry.Attempts(12), // last attempt should wait ~7min
	)
}
