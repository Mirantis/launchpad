package config

import (
	"testing"

	"github.com/k0sproject/rig"
	"github.com/stretchr/testify/require"
)

func TestHostSwarmAddress(t *testing.T) {
	h := Host{
		Connection: rig.Connection{
			SSH: &rig.SSH{
				Address: "1.2.3.4",
			},
		},
		Metadata: &HostMetadata{
			InternalAddress: "1.2.3.4",
		},
	}

	require.Equal(t, "1.2.3.4:2377", h.SwarmAddress())

	h = Host{
		Connection: rig.Connection{
			WinRM: &rig.WinRM{
				Address: "10.0.0.1",
			},
		},
		Metadata: &HostMetadata{
			InternalAddress: "1.2.3.4",
		},
	}

	require.Equal(t, "1.2.3.4:2377", h.SwarmAddress())
}

func TestHostSwarmAddressOverride(t *testing.T) {
	// When SwarmAddressOverride is set it takes precedence over InternalAddress.
	h := Host{
		Connection: rig.Connection{
			SSH: &rig.SSH{Address: "172.19.121.30"},
		},
		SwarmAddressOverride: "172.19.121.30",
		Metadata: &HostMetadata{
			InternalAddress: "192.168.1.10",
		},
	}
	require.Equal(t, "172.19.121.30:2377", h.SwarmAddress())
}

func TestHostSwarmAddressOverrideEmpty(t *testing.T) {
	// An empty SwarmAddressOverride falls back to InternalAddress.
	h := Host{
		Connection: rig.Connection{
			SSH: &rig.SSH{Address: "172.19.121.30"},
		},
		SwarmAddressOverride: "",
		Metadata: &HostMetadata{
			InternalAddress: "192.168.1.10",
		},
	}
	require.Equal(t, "192.168.1.10:2377", h.SwarmAddress())
}

func TestHostAddress(t *testing.T) {
	h := Host{
		Connection: rig.Connection{
			SSH: &rig.SSH{
				Address: "1.2.3.4",
			},
		},
	}

	require.Equal(t, "1.2.3.4", h.Address())
}
