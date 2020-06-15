package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var hosts = Hosts{
	{
		Address: "man1",
		Role:    "manager",
	},
	{
		Address: "man2",
		Role:    "manager",
	},
	{
		Address: "work1",
		Role:    "worker",
	},
}

func TestFilter(t *testing.T) {
	managers := hosts.Filter(func(h *Host) bool {
		return h.Role == "manager"
	})

	workers := hosts.Filter(func(h *Host) bool {
		return h.Role == "worker"
	})

	require.Len(t, managers, 2)
	require.Equal(t, managers[0].Role, "manager")
	require.Equal(t, managers[1].Role, "manager")

	require.Len(t, workers, 1)
	require.Equal(t, workers[0].Role, "worker")
}

func TestFind(t *testing.T) {
	require.Equal(t, hosts.Find(func(h *Host) bool { return h.Address == "man2" }), hosts[1])
	require.Nil(t, hosts.Find(func(h *Host) bool { return h.Address == "man3" }))
}

func TestIndex(t *testing.T) {
	require.Equal(t, hosts.Index(func(h *Host) bool { return h.Address == "man2" }), 1)
	require.Equal(t, hosts.Index(func(h *Host) bool { return h.Address == "man3" }), -1)
}

func TestIndexAll(t *testing.T) {
	matches := hosts.IndexAll(func(h *Host) bool { return h.Role == "manager" })
	require.Len(t, matches, 2)
	require.Equal(t, hosts[matches[0]], hosts[0])
	require.Equal(t, hosts[matches[1]], hosts[1])
	noMatches := hosts.IndexAll(func(h *Host) bool { return h.Role == "foo" })
	require.Len(t, noMatches, 0)
}

func TestMap(t *testing.T) {
	addresses := hosts.Map(func(h *Host) interface{} { return h.Address })
	require.Len(t, addresses, 3)
	require.Equal(t, addresses[0], "man1")
	require.Equal(t, addresses[1], "man2")
}

func TestMapString(t *testing.T) {
	addresses := hosts.MapString(func(h *Host) string { return h.Address })
	require.Len(t, addresses, 3)
	require.Equal(t, addresses[0], "man1")
	require.Equal(t, addresses[1], "man2")
}

func TestInclude(t *testing.T) {
	require.True(t, hosts.Include(func(h *Host) bool { return h.Role == "worker" }))
	require.False(t, hosts.Include(func(h *Host) bool { return h.Role == "foo" }))
}

func TestCount(t *testing.T) {
	require.Equal(t, hosts.Count(func(h *Host) bool { return h.Role == "manager" }), 2)
	require.Equal(t, hosts.Count(func(h *Host) bool { return h.Role == "worker" }), 1)
	require.Equal(t, hosts.Count(func(h *Host) bool { return h.Role == "foo" }), 0)
}

func ExampleHosts_Filter() {
	hosts := Hosts{
		{Role: "manager"},
		{Role: "worker"},
	}

	managers := hosts.Filter(func(h *Host) bool {
		return h.Role == "manager"
	})

	managers[0].Connect()
}
