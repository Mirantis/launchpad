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
