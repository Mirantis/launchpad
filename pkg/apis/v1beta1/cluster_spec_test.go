package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClusterSpecWebURLWithoutSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			&Host{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{},
	}
	require.Equal(t, "https://192.168.1.2", spec.WebURL())
}

func TestClusterSpecWebURLWithSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			&Host{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com"},
		},
	}
	require.Equal(t, "https://ucp.acme.com", spec.WebURL())
}

func TestClusterSpecWebURLWithSanSpace(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			&Host{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san ucp.acme.com"},
		},
	}
	require.Equal(t, "https://ucp.acme.com", spec.WebURL())
}

func TestClusterSpecWebURLWithMultiSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			&Host{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com", "--san=admin.acme.com"},
		},
	}
	require.Equal(t, "https://ucp.acme.com", spec.WebURL())
}
