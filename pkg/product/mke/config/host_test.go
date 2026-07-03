package config

import (
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/k0sproject/rig/v2/protocol/winrm"
	"github.com/stretchr/testify/require"
)

func TestHostSwarmAddress(t *testing.T) {
	h := Host{
		CompositeConfig: rig.CompositeConfig{
			SSH: &ssh.Config{
				Address: "1.2.3.4",
			},
		},
		Metadata: &HostMetadata{
			InternalAddress: "1.2.3.4",
		},
	}

	require.Equal(t, "1.2.3.4:2377", h.SwarmAddress())

	h = Host{
		CompositeConfig: rig.CompositeConfig{
			WinRM: &winrm.Config{
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
		CompositeConfig: rig.CompositeConfig{
			SSH: &ssh.Config{Address: "172.19.121.30"},
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
		CompositeConfig: rig.CompositeConfig{
			SSH: &ssh.Config{Address: "172.19.121.30"},
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
		CompositeConfig: rig.CompositeConfig{
			SSH: &ssh.Config{
				Address: "1.2.3.4",
			},
		},
	}

	require.Equal(t, "1.2.3.4", h.Address())
}
