package configurer

import (
	"testing"

	"github.com/k0sproject/rig/v2/sh"
	"github.com/stretchr/testify/require"
)

func TestIsSafeRemotePath(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		path string
		safe bool
	}{
		{path: "/var/lib/docker", safe: true},
		{path: "/var/lib/docker-1.2.3", safe: true},
		{path: "/var/run/docker.sock", safe: true},
		{path: "/var/lib/docker/cri-dockerd", safe: true},
		{path: "var/lib/docker", safe: false},       // not absolute
		{path: "./var/lib/docker", safe: false},     // not absolute
		{path: "/var/lib/../docker", safe: false},   // traversal
		{path: "/var/lib/docker/..", safe: false},   // traversal tail
		{path: "/var/lib//docker", safe: false},     // empty segment
		{path: "/", safe: false},                    // root itself
		{path: "", safe: false},                     // empty
		{path: "/var/lib/$(rm -rf /)", safe: false}, // command substitution
		{path: "/var/lib/; rm -rf /", safe: false},  // command separator
		{path: "/var/lib/my docker", safe: false},   // space
		{path: "/var/lib/d;rm", safe: false},        // separator
		{path: "`rm -rf /`", safe: false},           // backticks, not absolute
		{path: "/tmp/x\nrm -rf /", safe: false},     // newline
	} {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.safe, isSafeRemotePath(tc.path), "path %q", tc.path)
		})
	}
}

// TestResetCommandsQuoteHostPaths pins the sink contract: sh.Command quotes
// arguments by construction, so even a path that slipped past
// isSafeRemotePath arrives as a single argument instead of executing.
func TestResetCommandsQuoteHostPaths(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		`rm -rf /var/lib/docker`,
		sh.Command("rm", "-rf", "/var/lib/docker"),
	)
	require.Contains(t,
		sh.Command("umount", "/var/run/docker/netns/default"),
		`/var/run/docker/netns/default`,
	)
	// A metacharacter-bearing path - the shape isSafeRemotePath rejects
	// upstream - is quoted rather than executed.
	require.Equal(t,
		`rm -rf '/var/lib/$(id)'`,
		sh.Command("rm", "-rf", "/var/lib/$(id)"),
	)
}
