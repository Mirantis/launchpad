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
		// Location, not just syntax: these are all syntactically perfect
		// absolute paths that must never reach a sudo'd rm -rf.
		{path: "/etc", safe: false},                             // outside the allowlist
		{path: "/usr/share", safe: false},                       // outside the allowlist
		{path: "/home/ubuntu/.local/share/docker", safe: false}, // home tree, rootless not detected
		{path: "/.", safe: false},                               // non-canonical, derives an empty parent
		{path: "/var/run/docker/", safe: false},                 // trailing slash corrupts the parent derivation
		{path: "/var/lib/", safe: false},                        // trailing slash on the allowlist root
		{path: "/var/lib", safe: false},                         // the allowlist root itself is not deletable
		{path: "/var/run", safe: false},                         // the allowlist root itself is not deletable
		{path: "/run", safe: false},                             // the allowlist root itself is not deletable
		{path: "/var/libfoo/docker", safe: false},               // prefix match must be segment aware
		{path: "/run/docker", safe: true},
		{path: "/run/docker/netns", safe: true},
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

// TestIsSafeRemotePathRootlessHomeTree covers the rootless gap: a data-root
// under the user's home tree is only admitted when rootless detection actually
// resolved that home tree. The third case is the one that rots silently - when
// the home tree cannot be resolved the guard must degrade to the rootful
// allowlist rather than guessing a home directory.
func TestIsSafeRemotePathRootlessHomeTree(t *testing.T) {
	t.Parallel()

	const rootlessRoot = "/home/ubuntu/.local/share/docker"

	t.Run("rootless detected admits the home tree", func(t *testing.T) {
		t.Parallel()

		require.True(t, isSafeRemotePath(rootlessRoot, "/home/ubuntu"))
	})

	t.Run("rootless not detected rejects the home tree", func(t *testing.T) {
		t.Parallel()

		require.False(t, isSafeRemotePath(rootlessRoot))
	})

	t.Run("unresolvable home falls back to the rootful allowlist", func(t *testing.T) {
		t.Parallel()

		// No extra prefix is supplied, which is what CleanupLingeringMCR does
		// when rootlessHomeTree returns "".
		require.False(t, isSafeRemotePath(rootlessRoot))
		require.True(t, isSafeRemotePath("/var/lib/docker"))
	})

	t.Run("extra prefix does not admit paths outside it", func(t *testing.T) {
		t.Parallel()

		require.False(t, isSafeRemotePath("/home/other/.local/share/docker", "/home/ubuntu"))
		require.False(t, isSafeRemotePath("/etc", "/home/ubuntu"))
		// Segment aware: /home/ubuntu2 is not inside /home/ubuntu.
		require.False(t, isSafeRemotePath("/home/ubuntu2/docker", "/home/ubuntu"))
	})
}

// TestRootlessHomeTree pins which host-reported XDG_CONFIG_HOME values are
// allowed to widen the delete allowlist. The value is host-controlled, so
// taking its parent unchecked would let a host nominate any directory.
func TestRootlessHomeTree(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		xdgConfigHome string
		want          string
	}{
		{xdgConfigHome: "/home/ubuntu/.config", want: "/home/ubuntu"},
		{xdgConfigHome: "/root/.config", want: "/root"},
		{xdgConfigHome: "/home/ubuntu/.config/", want: ""}, // non-canonical
		{xdgConfigHome: "/etc/x", want: ""},                // would nominate /etc
		{xdgConfigHome: "/var/lib/docker/x", want: ""},     // not a home config dir
		{xdgConfigHome: "/home/.config", want: ""},         // no user segment
		{xdgConfigHome: "/.config", want: ""},              // would nominate /
		{xdgConfigHome: "/", want: ""},                     // would nominate /
		{xdgConfigHome: "", want: ""},                      // unset
		{xdgConfigHome: "/home/u/$(id)", want: ""},         // command substitution
		{xdgConfigHome: "/home/u/../../etc/x", want: ""},   // traversal
	} {
		t.Run(tc.xdgConfigHome, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, rootlessHomeTree(tc.xdgConfigHome))
		})
	}
}
