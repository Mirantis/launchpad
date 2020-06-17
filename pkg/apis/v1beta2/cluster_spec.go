package v1beta2

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts  Hosts        `yaml:"hosts" validate:"required,dive,min=1"`
	Ucp    UcpConfig    `yaml:"ucp"`
	Engine EngineConfig `yaml:"engine"`
}

// Workers filters only the workers from the cluster config
func (c *ClusterSpec) Workers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "worker" })
}

// Managers filters only the manager nodes from the cluster config
func (c *ClusterSpec) Managers() Hosts {
	return c.Hosts.Filter(func(h *Host) bool { return h.Role == "manager" })
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

// WebURL returns an URL to web UI
func (c *ClusterSpec) WebURL() string {
	address := c.Managers()[0].Address
	san := c.Ucp.getInstallFlagValue("--san")
	if san != "" {
		address = san
	}

	return fmt.Sprintf("https://%s", address)
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *ClusterSpec) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type yclusterspec ClusterSpec
	yc := (*yclusterspec)(c)
	c.Engine = NewEngineConfig()
	c.Ucp = NewUcpConfig()

	if err := unmarshal(yc); err != nil {
		return err
	}

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
