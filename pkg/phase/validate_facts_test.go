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
	require.EqualError(t, phase.validateUCPVersionJump(config), "can't upgrade UCP directly from 3.1.1 to 3.3.3-tp9 - need to upgrade to 3.2 first.")
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
			Dtr: api.DtrConfig{
				Metadata: &api.DtrMetadata{
					Installed:        true,
					InstalledVersion: "2.6.4",
				},
				Version: "2.8.4",
			},
		},
	}
	require.EqualError(t, phase.validateDTRVersionJump(config), "can't upgrade DTR directly from 2.6.4 to 2.8.4 - need to upgrade to 2.7 first.")
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
