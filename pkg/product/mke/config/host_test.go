package config

import (
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol/ssh"
	"github.com/k0sproject/rig/v2/protocol/winrm"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
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

// TestHostWinRMHTTPSPortDefault covers the rig v0 -> v2 compatibility shim in
// Host.UnmarshalYAML. creasty/defaults applies rig's winRM port struct-tag default
// (5985) before rig can derive it, and rig v2 only derives the port when it is zero.
// Without the shim, `useHTTPS: true` with no explicit port would try TLS against the
// plaintext port. See PRODENG-3594.
func TestHostWinRMHTTPSPortDefault(t *testing.T) {
	for _, tc := range []struct {
		name     string
		yaml     string
		expected int
	}{
		{
			name:     "useHTTPS with no explicit port is bumped to the TLS port",
			yaml:     "winRM:\n  address: 10.0.0.1\n  useHTTPS: true\n",
			expected: 5986,
		},
		{
			name:     "useHTTPS with the plaintext port is bumped, matching rig v0",
			yaml:     "winRM:\n  address: 10.0.0.1\n  useHTTPS: true\n  port: 5985\n",
			expected: 5986,
		},
		{
			name:     "without useHTTPS the plaintext port is left alone",
			yaml:     "winRM:\n  address: 10.0.0.1\n",
			expected: 5985,
		},
		{
			name:     "an explicit TLS port is preserved",
			yaml:     "winRM:\n  address: 10.0.0.1\n  useHTTPS: true\n  port: 5986\n",
			expected: 5986,
		},
		{
			name:     "a custom port is never rewritten",
			yaml:     "winRM:\n  address: 10.0.0.1\n  useHTTPS: true\n  port: 15986\n",
			expected: 15986,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := &Host{}
			require.NoError(t, yaml.Unmarshal([]byte(tc.yaml), h))
			require.NotNil(t, h.WinRM)
			require.Equal(t, tc.expected, h.WinRM.Port)
		})
	}
}
