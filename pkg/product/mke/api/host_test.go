package api

import (
	"testing"

	common "github.com/Mirantis/mcc/pkg/product/common/api"
	"github.com/stretchr/testify/require"
)

func TestHost_SwarmAddress(t *testing.T) {

	h := Host{
		ConnectableHost: common.ConnectableHost{Address: "1.2.3.4"},
		Metadata: &HostMetadata{
			InternalAddress: "1.2.3.4",
		},
	}

	require.Equal(t, "1.2.3.4:2377", h.SwarmAddress())
}
