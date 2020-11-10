package phase

import (
	"strings"
	"testing"

	"github.com/Mirantis/mcc/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestValidateFactsUCPVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{
					Installed:        true,
					InstalledVersion: "3.1.1",
				},
				Version: "3.3.3-tp9",
			},
		},
	}
	require.EqualError(t, phase.validateUCPVersionJump(config), "can't upgrade UCP directly from 3.1.1 to 3.3.3-tp9 - need to upgrade to 3.2 first")
}

func TestValidateFactsUCPVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{
					Installed:        true,
					InstalledVersion: "3.3.3-tp9",
				},
				Version: "3.2.8",
			},
		},
	}
	require.EqualError(t, phase.validateUCPVersionJump(config), "can't downgrade UCP 3.3.3-tp9 to 3.2.8")
}

func TestValidateFactsUCPVersionJumpSuccess(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{
					Installed:        true,
					InstalledVersion: "3.1.1",
				},
				Version: "3.2.8",
			},
		},
	}
	require.NoError(t, phase.validateUCPVersionJump(config))
}

func TestValidateFactsDTRVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Dtr: &api.DtrConfig{
				Metadata: &api.DtrMetadata{
					Installed:        true,
					InstalledVersion: "2.6.4",
				},
				Version: "2.8.4",
			},
		},
	}
	require.EqualError(t, phase.validateDTRVersionJump(config), "can't upgrade DTR directly from 2.6.4 to 2.8.4 - need to upgrade to 2.7 first")
}

func TestValidateFactsDTRVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Dtr: &api.DtrConfig{
				Metadata: &api.DtrMetadata{
					Installed:        true,
					InstalledVersion: "2.8.4",
				},
				Version: "2.7.6",
			},
		},
	}
	require.EqualError(t, phase.validateDTRVersionJump(config), "can't downgrade DTR 2.8.4 to 2.7.6")
}

func TestValidateFactsDTRVersionJumpSuccess(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Dtr: &api.DtrConfig{
				Metadata: &api.DtrMetadata{
					Installed:        true,
					InstalledVersion: "2.6.8",
				},
				Version: "2.7.1",
			},
		},
	}
	require.NoError(t, phase.validateDTRVersionJump(config))
}

func TestValidateFactsValidateDataPlane(t *testing.T) {
	phase := ValidateFacts{}
	conf := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				InstallFlags: []string{
					"--foo",
					"--calico-vxlan=true",
				},
				Metadata: &api.UcpMetadata{
					Installed: true,
					VXLAN:     false,
				},
			},
		},
	}

	// Test meta-vxlan: false, --calico-vxlan=true
	require.EqualError(t, phase.validateDataPlane(conf), "calico configured with IPIP, can't automatically change to VXLAN")

	// Test meta-vxlan: false, --calico-vxlan (should evaluate to true)
	conf.Spec.Ucp.InstallFlags = []string{
		"--calico-vxlan",
	}
	require.EqualError(t, phase.validateDataPlane(conf), "calico configured with IPIP, can't automatically change to VXLAN")

	// Test with meta-vxlan: true, --calico-vxlan true
	conf.Spec.Ucp.Metadata.VXLAN = true
	require.NoError(t, phase.validateDataPlane(conf))

	// Test with meta-vxlan: true, --calico-vxlan false
	conf.Spec.Ucp.InstallFlags = []string{
		"--calico-vxlan=false",
	}
	require.EqualError(t, phase.validateDataPlane(conf), "calico configured with VXLAN, can't automatically change to IPIP")

	// Test with meta-vxlan: false, --calico-vxlan false
	conf.Spec.Ucp.Metadata.VXLAN = false
	require.NoError(t, phase.validateDataPlane(conf))
}

func TestValidateFactsPopulateSan(t *testing.T) {
	phase := ValidateFacts{}
	conf := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: api.Hosts{
				&api.Host{Address: "10.0.0.1", Role: "manager"},
				&api.Host{Address: "10.0.0.2", Role: "manager"},
				&api.Host{Address: "10.0.0.3", Role: "worker"},
			},
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{},
				InstallFlags: api.Flags{
					"--foo",
				},
			},
		},
	}
	phase.Prepare(conf)
	phase.Run()
	var sans []string

	for _, v := range conf.Spec.Ucp.InstallFlags {
		if strings.HasPrefix(v, "--san") {
			sans = append(sans, v)
		}
	}

	require.Len(t, conf.Spec.Ucp.InstallFlags, 3)
	require.Len(t, sans, 2)

	require.Equal(t, "--san=10.0.0.1", sans[0])
	require.Equal(t, "--san=10.0.0.2", sans[1])
}

func TestValidateFactsDontPopulateSan(t *testing.T) {
	phase := ValidateFacts{}
	conf := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Hosts: api.Hosts{
				&api.Host{Address: "10.0.0.1", Role: "manager"},
				&api.Host{Address: "10.0.0.2", Role: "manager"},
				&api.Host{Address: "10.0.0.3", Role: "worker"},
			},
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{},
				InstallFlags: api.Flags{
					"--foo",
					"--san foofoo",
				},
			},
		},
	}
	phase.Prepare(conf)
	phase.Run()
	var sans []string

	for _, v := range conf.Spec.Ucp.InstallFlags {
		if strings.HasPrefix(v, "--san") {
			sans = append(sans, v)
		}
	}

	require.Len(t, sans, 1)
	require.Equal(t, "--san foofoo", sans[0])
}
