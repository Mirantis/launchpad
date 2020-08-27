package phase

import (
	"testing"

	api "github.com/Mirantis/mcc/pkg/apis/v1beta3"
	"github.com/stretchr/testify/require"
)

func TestValidateHostsValidateDataPlane(t *testing.T) {
	phase := ValidateHosts{}
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
