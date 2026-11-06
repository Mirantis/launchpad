package configurer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLicenseFilePath pins where the MCR license lands for a host-reported
// docker root. MCR reads its subscription from its own data-root, so on a
// rootless host the license has to follow the home data-root: falling back to
// /var/lib/docker there leaves the daemon unlicensed rather than just moving
// the file. The same path on a rootful host must still be rejected.
func TestLicenseFilePath(t *testing.T) {
	t.Parallel()

	const rootlessRoot = "/home/ubuntu/.local/share/docker"

	// The prefixes a rootless host yields, derived the way
	// resolveDockerDaemonPath derives them.
	rootlessPrefixes := []string{rootlessHomeTree("/home/ubuntu/.config")}
	require.Equal(t, []string{"/home/ubuntu"}, rootlessPrefixes)

	for _, tc := range []struct {
		name          string
		reportedRoot  string
		extraPrefixes []string
		want          string
		wantTrusted   bool
	}{
		{
			name:          "rootless detected writes under the home data-root",
			reportedRoot:  rootlessRoot,
			extraPrefixes: rootlessPrefixes,
			want:          "/home/ubuntu/.local/share/docker/docker.lic",
			wantTrusted:   true,
		},
		{
			name:         "rootful host reporting the same path uses the default",
			reportedRoot: rootlessRoot,
			want:         "/var/lib/docker/docker.lic",
		},
		{
			name:          "unresolvable home falls back to the default",
			reportedRoot:  rootlessRoot,
			extraPrefixes: nil, // rootlessHomeTree returned "" for this host
			want:          "/var/lib/docker/docker.lic",
		},
		{
			name:          "rootless prefix does not admit another user's tree",
			reportedRoot:  "/home/other/.local/share/docker",
			extraPrefixes: rootlessPrefixes,
			want:          "/var/lib/docker/docker.lic",
		},
		{
			name:         "rootful data-root is honoured",
			reportedRoot: "/var/lib/docker-alt",
			want:         "/var/lib/docker-alt/docker.lic",
			wantTrusted:  true,
		},
		{
			name:         "empty root does not write to a relative path",
			reportedRoot: "",
			want:         "/var/lib/docker/docker.lic",
		},
		{
			name:         "host-chosen location is rejected",
			reportedRoot: "/etc",
			want:         "/var/lib/docker/docker.lic",
		},
		{
			name:          "metacharacters are rejected even under the home tree",
			reportedRoot:  "/home/ubuntu/$(id)",
			extraPrefixes: rootlessPrefixes,
			want:          "/var/lib/docker/docker.lic",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			licPath, trusted := licenseFilePath(tc.reportedRoot, tc.extraPrefixes)
			require.Equal(t, tc.want, licPath)
			require.Equal(t, tc.wantTrusted, trusted)
		})
	}
}
