package configurer

import (
	"testing"

	"github.com/Mirantis/launchpad/pkg/constant"
	"github.com/stretchr/testify/require"
)

// TestWindowsLicenseFilePath pins that the remote Windows license path is
// built with the target's separator whatever the control machine runs.
// filepath.Join would produce "C:\ProgramData\Docker/docker.lic" from Linux
// and the expectation here would then depend on the machine running the test.
func TestWindowsLicenseFilePath(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		reportedRoot string
		want         string
	}{
		{
			name:         "default root",
			reportedRoot: constant.WindowsDefaultDockerRoot,
			want:         `C:\ProgramData\Docker\docker.lic`,
		},
		{
			name:         "host reported root",
			reportedRoot: `D:\docker-data`,
			want:         `D:\docker-data\docker.lic`,
		},
		{
			name:         "trailing backslash is not doubled",
			reportedRoot: `D:\docker-data\`,
			want:         `D:\docker-data\docker.lic`,
		},
		{
			name:         "trailing forward slash",
			reportedRoot: "D:/docker-data/",
			want:         `D:/docker-data\docker.lic`,
		},
		{
			name:         "empty root does not write to the drive root",
			reportedRoot: "",
			want:         `C:\ProgramData\Docker\docker.lic`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, windowsLicenseFilePath(tc.reportedRoot))
		})
	}
}
