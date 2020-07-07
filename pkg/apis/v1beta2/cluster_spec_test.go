package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUCPClusterSpecWebURLWithoutSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{},
	}
	expected := &WebUrls{
		Ucp: "https://192.168.1.2",
		Dtr: "",
	}
	actual := spec.WebURLs()
	require.Equal(t, expected, actual)
}

func TestUCPClusterSpecWebURLWithSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com"},
		},
	}
	expected := &WebUrls{
		Ucp: "https://ucp.acme.com",
		Dtr: "",
	}
	actual := spec.WebURLs()
	require.Equal(t, expected, actual)
}

func TestUCPClusterSpecWebURLWithSanSpace(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san ucp.acme.com"},
		},
	}
	expected := &WebUrls{
		Ucp: "https://ucp.acme.com",
		Dtr: "",
	}
	actual := spec.WebURLs()
	require.Equal(t, expected, actual)
}

func TestUCPClusterSpecWebURLWithMultiSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com", "--san=admin.acme.com"},
		},
	}
	expected := &WebUrls{
		Ucp: "https://ucp.acme.com",
		Dtr: "",
	}
	actual := spec.WebURLs()
	require.Equal(t, expected, actual)
}

func TestUCPClusterSpecWebURLWithDTRWebURLWithoutExternalURL(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{},
		Dtr: DtrConfig{
			Metadata: &DtrMetadata{
				DtrLeaderAddress: "192.168.1.3",
			},
		},
	}
	expected := &WebUrls{
		Ucp: "https://192.168.1.2",
		Dtr: "https://192.168.1.3",
	}
	actual := spec.WebURLs()
	require.Equal(t, expected, actual)
}

func TestUCPClusterSpecWebURLWithDTRWebURLWithExternalURL(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{Address: "192.168.1.2", Role: "manager"},
			{Address: "192.168.1.3", Role: "dtr"},
		},
		Ucp: UcpConfig{
			InstallFlags: []string{"--san=ucp.acme.com"},
		},
		Dtr: DtrConfig{
			InstallFlags: []string{"--dtr-external-url dtr.acme.com"},
		},
	}
	expected := &WebUrls{
		Ucp: "https://ucp.acme.com",
		Dtr: "https://dtr.acme.com",
	}
	actual := spec.WebURLs()
	require.Equal(t, expected, actual)
}
