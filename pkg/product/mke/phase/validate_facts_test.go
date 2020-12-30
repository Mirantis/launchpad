package phase

import (
	"strings"
	"testing"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
	"github.com/stretchr/testify/require"
)

func TestValidateFactsMKEVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			MKE: api.MKEConfig{
				Metadata: &api.MKEMetadata{
					Installed:        true,
					InstalledVersion: "3.1.1",
				},
				Version: "3.3.3-tp9",
			},
		},
	}
	require.EqualError(t, phase.validateMKEVersionJump(), "can't upgrade MKE directly from 3.1.1 to 3.3.3-tp9 - need to upgrade to 3.2 first")
}

func TestValidateFactsMKEVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			MKE: api.MKEConfig{
				Metadata: &api.MKEMetadata{
					Installed:        true,
					InstalledVersion: "3.3.3-tp9",
				},
				Version: "3.2.8",
			},
		},
	}
	require.EqualError(t, phase.validateMKEVersionJump(), "can't downgrade MKE 3.3.3-tp9 to 3.2.8")
}

func TestValidateFactsMKEVersionJumpSuccess(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			MKE: api.MKEConfig{
				Metadata: &api.MKEMetadata{
					Installed:        true,
					InstalledVersion: "3.1.1",
				},
				Version: "3.2.8",
			},
		},
	}
	require.NoError(t, phase.validateMKEVersionJump())
}

func TestValidateFactsMSRVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: []*api.Host{
				{Role: "msr", MSRMetadata: &api.MSRMetadata{
					Installed:        true,
					InstalledVersion: "2.6.4",
				}},
			},
			MSR: &api.MSRConfig{
				Version: "2.8.4",
			},
		},
	}
	require.EqualError(t, phase.validateMSRVersionJump(), "can't upgrade MSR directly from 2.6.4 to 2.8.4 - need to upgrade to 2.7 first")
}
func TestValidateFactsMSRVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: []*api.Host{
				{Role: "msr", MSRMetadata: &api.MSRMetadata{
					Installed:        true,
					InstalledVersion: "2.8.4",
				}},
			},
			MSR: &api.MSRConfig{
				Version: "2.7.6",
			},
		},
	}
	require.EqualError(t, phase.validateMSRVersionJump(), "can't downgrade MSR 2.8.4 to 2.7.6")
}

func TestValidateFactsMSRVersionJumpSuccess(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: []*api.Host{
				{Role: "msr", MSRMetadata: &api.MSRMetadata{
					Installed:        true,
					InstalledVersion: "2.6.8",
				}},
			},
			MSR: &api.MSRConfig{
				Version: "2.7.1",
			},
		},
	}
	require.NoError(t, phase.validateMSRVersionJump())
}

func TestValidateFactsValidateDataPlane(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			MKE: api.MKEConfig{
				InstallFlags: []string{
					"--foo",
					"--calico-vxlan=true",
				},
				Metadata: &api.MKEMetadata{
					Installed: true,
					VXLAN:     false,
				},
			},
		},
	}

	// Test meta-vxlan: false, --calico-vxlan=true
	require.EqualError(t, phase.validateDataPlane(), "calico configured with IPIP, can't automatically change to VXLAN")

	// Test meta-vxlan: false, --calico-vxlan (should evaluate to true)
	phase.Config.Spec.MKE.InstallFlags = []string{
		"--calico-vxlan",
	}
	require.EqualError(t, phase.validateDataPlane(), "calico configured with IPIP, can't automatically change to VXLAN")

	// Test with meta-vxlan: true, --calico-vxlan true
	phase.Config.Spec.MKE.Metadata.VXLAN = true
	require.NoError(t, phase.validateDataPlane())

	// Test with meta-vxlan: true, --calico-vxlan false
	phase.Config.Spec.MKE.InstallFlags = []string{
		"--calico-vxlan=false",
	}
	require.EqualError(t, phase.validateDataPlane(), "calico configured with VXLAN, can't automatically change to IPIP")

	// Test with meta-vxlan: false, --calico-vxlan false
	phase.Config.Spec.MKE.Metadata.VXLAN = false
	require.NoError(t, phase.validateDataPlane())
}

func TestValidateFactsPopulateSan(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: api.Hosts{
				&api.Host{ConnectableHost: common.ConnectableHost{Address: "10.0.0.1"}, Role: "manager"},
				&api.Host{ConnectableHost: common.ConnectableHost{Address: "10.0.0.2"}, Role: "manager"},
				&api.Host{ConnectableHost: common.ConnectableHost{Address: "10.0.0.3"}, Role: "worker"},
			},
			MKE: api.MKEConfig{
				Metadata: &api.MKEMetadata{},
				InstallFlags: common.Flags{
					"--foo",
				},
			},
		},
	}
	phase.Run()
	var sans []string

	for _, v := range phase.Config.Spec.MKE.InstallFlags {
		if strings.HasPrefix(v, "--san") {
			sans = append(sans, v)
		}
	}

	require.Len(t, phase.Config.Spec.MKE.InstallFlags, 3)
	require.Len(t, sans, 2)

	require.Equal(t, "--san=10.0.0.1", sans[0])
	require.Equal(t, "--san=10.0.0.2", sans[1])
}

func TestValidateFactsDontPopulateSan(t *testing.T) {
	phase := ValidateFacts{}
	phase.Config = &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: api.Hosts{
				&api.Host{ConnectableHost: common.ConnectableHost{Address: "10.0.0.1"}, Role: "manager"},
				&api.Host{ConnectableHost: common.ConnectableHost{Address: "10.0.0.2"}, Role: "manager"},
				&api.Host{ConnectableHost: common.ConnectableHost{Address: "10.0.0.3"}, Role: "worker"},
			},
			MKE: api.MKEConfig{
				Metadata: &api.MKEMetadata{},
				InstallFlags: common.Flags{
					"--foo",
					"--san foofoo",
				},
			},
		},
	}
	phase.Run()
	var sans []string

	for _, v := range phase.Config.Spec.MKE.InstallFlags {
		if strings.HasPrefix(v, "--san") {
			sans = append(sans, v)
		}
	}

	require.Len(t, sans, 1)
	require.Equal(t, "--san foofoo", sans[0])
}
