package swarm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The inputs below are the verbatim output of defaultAddrPoolCmd captured from
// MCR 29.1.3, so a change in docker's rendering fails these rather than silently
// mis-parsing on a live cluster.
func TestParseDefaultAddrPool(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  string
		want []string
	}{
		{
			// Not in a swarm: the pool is unknown rather than defaulted, so
			// callers can tell a greenfield host from a configured one.
			name: "host is not in a swarm",
			out:  "inactive|",
			want: nil,
		},
		{
			// A swarm created without --default-addr-pool reports no pool but is
			// running on docker's built-in default.
			name: "swarm without an explicit pool uses the built-in default",
			out:  "active|",
			want: []string{DefaultAddrPoolFallback},
		},
		{
			name: "swarm with one pool",
			out:  "active|10.10.0.0/16 ",
			want: []string{"10.10.0.0/16"},
		},
		{
			// --default-addr-pool may be repeated; every pool has to survive.
			name: "swarm with repeated pools",
			out:  "active|10.10.0.0/16 172.31.0.0/16 ",
			want: []string{"10.10.0.0/16", "172.31.0.0/16"},
		},
		{
			name: "pending state is not treated as a swarm",
			out:  "pending|",
			want: nil,
		},
		{
			// A locked manager cannot be read from, and reporting the fallback
			// would assert a pool that was never observed.
			name: "locked state is not treated as a swarm",
			out:  "locked|",
			want: nil,
		},
		{
			name: "surrounding whitespace is tolerated",
			out:  "  active |  10.10.0.0/16  ",
			want: []string{"10.10.0.0/16"},
		},
		{
			// Defensive: an unexpectedly empty read must not claim a pool.
			name: "empty output",
			out:  "",
			want: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, parseDefaultAddrPool(tc.out))
		})
	}
}
