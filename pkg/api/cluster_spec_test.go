package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUCPClusterSpecUcpURLWithoutSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{},
	}
	url, err := spec.UcpURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.2/", url.String())
}

func TestUCPClusterSpecUcpURLWithSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com"},
		},
		Dtr: &DtrConfig{},
	}

	url, err := spec.UcpURL()
	require.NoError(t, err)
	require.Equal(t, "https://ucp.acme.com/", url.String())
}

func TestUCPClusterSpecUcpURLWithMultiSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com", "--san=admin.acme.com"},
		},
	}
	url, err := spec.UcpURL()
	require.NoError(t, err)
	require.Equal(t, "https://ucp.acme.com/", url.String())
}

func TestUCPClusterSpecUcpURLWithNoDTRMetadata(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{},
	}

	url, err := spec.UcpURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.2/", url.String())
}

func TestUCPClusterSpecDtrURLWithNoDTRMetadata(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{},
	}

	url, err := spec.DtrURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.3/", url.String())
}

func TestUCPClusterSpecDtrURLWithNoDTRHostRoleButConfig(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{},
	}
	_, err := spec.DtrURL()
	require.Error(t, err)
}

func TestUCPClusterSpecDtrURLWithoutExternalURL(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{
			Metadata: &DtrMetadata{
				DtrLeaderAddress: "192.168.1.3",
			},
		},
	}
	url, err := spec.DtrURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.3/", url.String())
}

func TestUCPClusterSpecDtrURLWithExternalURL(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{
			InstallFlags: []string{"--dtr-external-url dtr.acme.com"},
		},
	}
	url, err := spec.DtrURL()
	require.NoError(t, err)
	require.Equal(t, "https://dtr.acme.com/", url.String())
}

func TestUCPClusterSpecDtrURLWithPort(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{
			InstallFlags: []string{"--replica-https-port 999"},
		},
	}
	url, err := spec.DtrURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.3:999/", url.String())
}

func TestUCPClusterSpecUcpURLWithPort(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--controller-port 999"},
		},
	}
	url, err := spec.UcpURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.2:999/", url.String())
}

func TestUCPClusterSpecUcpURLFromDtrUcpUrl(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: &DtrConfig{
			InstallFlags: []string{"--ucp-url ucp.acme.com:5555"},
		},
	}
	url, err := spec.UcpURL()
	require.NoError(t, err)
	require.Equal(t, "https://ucp.acme.com:5555/", url.String())
}
