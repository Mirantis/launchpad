package api

// ClusterSpec defines cluster spec
type ClusterSpec struct {
	Hosts Hosts     `yaml:"hosts" validate:"required,dive,min=1"`
	K0s   K0sConfig `yaml:"k0s,omitempty"`

	k0sLeader *Host
}

func (c *ClusterSpec) K0sLeader() *Host {
	if c.k0sLeader == nil {
		// Pick the first server that reports to be running and persist the choice
		for _, h := range c.Hosts {
			if h.Role == "server" && h.Metadata.K0sVersion != "" && h.InitSystem.ServiceIsRunning("k0s") {
				c.k0sLeader = h
			}
		}
	}

	// Still nil?  Fall back to first "server" host, do not persist selection.
	if c.k0sLeader == nil {
		for _, h := range c.Hosts {
			if h.Role == "server" {
				return h
			}
		}
	}

	return c.k0sLeader
}
