package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeReadyState(t *testing.T) {
	n := Node{
		Status: NodeStatus{
			State: NODE_READY_STATE,
		},
	}

	require.Equal(t, true, n.IsReady())
}

func TestNodeNotReadyState(t *testing.T) {
	n := Node{
		Status: NodeStatus{
			State: "pending",
		},
	}

	require.Equal(t, false, n.IsReady())
}
