package v1beta2

import (
	"fmt"
	"strings"

	"github.com/Mirantis/mcc/pkg/constant"
	"github.com/Mirantis/mcc/pkg/util"
	log "github.com/sirupsen/logrus"
)

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts  Hosts        `yaml:"hosts" validate:"required,dive,min=1"`
	Ucp    UcpConfig    `yaml:"ucp"`
	Dtr    DtrConfig    `yaml:"dtr"`
	Engine EngineConfig `yaml:"engine"`
}

// WebUrls holds admin web url strings for different products
type WebUrls struct {
	Ucp string
	Dtr string
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
		log.Debugf("%s: is the swarm leader", leader.Address)
		return leader
	}

	log.Debugf("did not find a real swarm manager, fallback to first manager host")
	return m.First()
}

// WebURLs returns a URL to web UI for both UCP and DTR
func (c *ClusterSpec) WebURLs() *WebUrls {
	ucpAddress := util.GetInstallFlagValue(c.Ucp.InstallFlags, "--san")
	if ucpAddress == "" {
		ucpAddress = c.Managers()[0].Address
	}

	ucpAddress = fmt.Sprintf("https://%s", ucpAddress)
	dtrAddress := c.buildDtrWebURL()

	return &WebUrls{
		Ucp: ucpAddress,
		Dtr: dtrAddress,
	}
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type yclusterspec ClusterSpec
	yc := (*yclusterspec)(c)
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
		log.Warnf("failed to get host's swarm leader status, probably not part of swarm")
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
		log.Debugf("%s: found DTR installed, using as leader", leader.Address)
		return leader
	}

	log.Debugf("did not find a DTR installation, falling back to first DTR host")
	return dtrs.First()
}

// IsCustomImageRepo checks if the config is using a custom image repo
func IsCustomImageRepo(imageRepo string) bool {
	return imageRepo != constant.ImageRepo
}

// buildDtrWebURL determines whether a web url for DTR should be built and if so
// returns one based on the DtrLeaderAddress or whether the user has provided
// the --dtr-external-url flag
func (c *ClusterSpec) buildDtrWebURL() string {
	for _, h := range c.Hosts {
		if h.Role == "dtr" {
			dtrAddress := util.GetInstallFlagValue(c.Dtr.InstallFlags, "--dtr-external-url")
			if dtrAddress != "" {
				return fmt.Sprintf("https://%s", dtrAddress)
			}

			return fmt.Sprintf("https://%s", c.Dtr.Metadata.DtrLeaderAddress)
		}
	}
	return ""
}
