package api

import (
	"testing"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/stretchr/testify/require"
)

func TestMKEClusterSpecMKEURLWithoutSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{},
	}
	url, err := spec.MKEURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.2/", url.String())
}

func TestMKEClusterSpecMKEURLWithSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
		},
		MKE: MKEConfig{
			InstallFlags: []string{"--san=mke.acme.com"},
		},
		MSR: &MSRConfig{},
	}

	url, err := spec.MKEURL()
	require.NoError(t, err)
	require.Equal(t, "https://mke.acme.com/", url.String())
}

func TestMKEClusterSpecMKEURLWithMultiSan(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
		},
		MKE: MKEConfig{
			InstallFlags: []string{"--san=mke.acme.com", "--san=admin.acme.com"},
		},
	}
	url, err := spec.MKEURL()
	require.NoError(t, err)
	require.Equal(t, "https://mke.acme.com/", url.String())
}

func TestMKEClusterSpecMKEURLWithNoMSRMetadata(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.3"}, Role: "msr"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{},
	}

	url, err := spec.MKEURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.2/", url.String())
}

func TestMKEClusterSpecMSRURLWithNoMSRMetadata(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.3"}, Role: "msr"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{},
	}

	url, err := spec.MSRURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.3/", url.String())
}

func TestMKEClusterSpecMSRURLWithNoMSRHostRoleButConfig(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{},
	}
	_, err := spec.MSRURL()
	require.Error(t, err)
}

func TestMKEClusterSpecMSRURLWithoutExternalURL(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.3"}, Role: "msr", MSRMetadata: &MSRMetadata{Installed: true}},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{},
	}
	url, err := spec.MSRURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.3/", url.String())
}

func TestMKEClusterSpecMSRURLWithExternalURL(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.3"}, Role: "msr"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{
			InstallFlags: []string{"--dtr-external-url msr.acme.com"},
		},
	}
	url, err := spec.MSRURL()
	require.NoError(t, err)
	require.Equal(t, "https://msr.acme.com/", url.String())
}

func TestMKEClusterSpecMSRURLWithPort(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.3"}, Role: "msr"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{
			InstallFlags: []string{"--replica-https-port 999"},
		},
	}
	url, err := spec.MSRURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.3:999/", url.String())
}

func TestMKEClusterSpecMKEURLWithPort(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
		},
		MKE: MKEConfig{
			InstallFlags: []string{"--controller-port 999"},
		},
	}
	url, err := spec.MKEURL()
	require.NoError(t, err)
	require.Equal(t, "https://192.168.1.2:999/", url.String())
}

func TestMKEClusterSpecMKEURLFromMSRMKEUrl(t *testing.T) {
	spec := ClusterSpec{
		Hosts: []*Host{
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.2"}, Role: "manager"},
			{ConnectableHost: common.ConnectableHost{Address: "192.168.1.3"}, Role: "msr"},
		},
		MKE: MKEConfig{},
		MSR: &MSRConfig{
			InstallFlags: []string{"--ucp-url mke.acme.com:5555"},
		},
	}
	url, err := spec.MKEURL()
	require.NoError(t, err)
	require.Equal(t, "https://mke.acme.com:5555/", url.String())
}
