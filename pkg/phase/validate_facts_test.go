package phase

import (
	"testing"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
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
	require.EqualError(t, phase.validateUCPVersionJump(config), "can't upgrade UCP directly from 3.1.1 to 3.3.3 - need to upgrade to 3.2 first")
}

func TestValidateFactsUCPVersionJumpDowngradeFail(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{
					Installed:        true,
					InstalledVersion: "3.3.3",
				},
				Version: "3.2.8",
			},
		},
	}
	require.EqualError(t, phase.validateUCPVersionJump(config), "can't downgrade UCP 3.3.3 to 3.2.8")
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

// The server reports version as 3.3.3 even when 3.3.3-tp10 is installed
func TestValidateFactsUCPVersionJumpSuccessBug(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				Metadata: &api.UcpMetadata{
					Installed:        true,
					InstalledVersion: "3.3.3",
				},
				Version: "3.3.3-tp10",
			},
		},
	}
	require.NoError(t, phase.validateUCPVersionJump(config))
}

func TestValidateFactsDTRVersionJumpFail(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Dtr: api.DtrConfig{
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
			Dtr: api.DtrConfig{
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
			Dtr: api.DtrConfig{
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

// The server reports version as 3.3.3 even when 3.3.3-tp10 is installed
func TestValidateFactsDTRVersionJumpSuccessBug(t *testing.T) {
	phase := ValidateFacts{}
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Dtr: api.DtrConfig{
				Metadata: &api.DtrMetadata{
					Installed:        true,
					InstalledVersion: "2.6.8",
				},
				Version: "2.6.8-beta1",
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
