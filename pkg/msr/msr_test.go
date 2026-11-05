package msr

import (
	"reflect"
	"sort"
	"testing"

	commonconfig "github.com/Mirantis/launchpad/pkg/product/common/config"
	mkeconfig "github.com/Mirantis/launchpad/pkg/product/mke/config"
	"github.com/stretchr/testify/require"
)

func TestPluckSharedInstallFlags(t *testing.T) {

	t.Run("Install flags are shared with join", func(t *testing.T) {
		installFlags := []string{
			"--replica-http-port 8000",
			"--replica-https-port 4443",
			"--ucp-insecure-tls",
			"--nfs-storage-url nfs://nfs.acme.com",
		}
		expectedJoinFlags := []string{
			"--replica-http-port 8000",
			"--replica-https-port 4443",
			"--ucp-insecure-tls",
		}
		actualJoinFlags := PluckSharedInstallFlags(installFlags, SharedInstallJoinFlags)
		sort.Strings(actualJoinFlags)
		sort.Strings(expectedJoinFlags)
		if !reflect.DeepEqual(actualJoinFlags, expectedJoinFlags) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expectedJoinFlags, actualJoinFlags)
		}
	})

	t.Run("Flags with multiple values can still be plucked", func(t *testing.T) {
		multiValueFlag := []string{
			"--fake-flag one two three",
		}
		testSharedFlags := []string{
			"--fake-flag",
		}
		expected := []string{
			"--fake-flag one two three",
		}
		actual := PluckSharedInstallFlags(multiValueFlag, testSharedFlags)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expected, actual)
		}
	})
}

func TestBuildMKEFlags(t *testing.T) {
	config := &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			MKE: mkeconfig.MKEConfig{
				AdminUsername: "admin",
				AdminPassword: "password1234",
				InstallFlags: []string{
					"--san ucp.acme.com",
				},
			},
			MSR: &mkeconfig.MSRConfig{},
		},
	}

	t.Run("MKE flags are built when --san is provided", func(t *testing.T) {
		actual := BuildMKEFlags(config)
		expected := commonconfig.Flags{
			"--ucp-url=\"ucp.acme.com\"",
			"--ucp-username=\"admin\"",
			"--ucp-password=\"password1234\"",
		}
		sort.Strings(actual)
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expected, actual)
		}
	})
}

func TestFormatReplicaID(t *testing.T) {
	require.Equal(t, "000000000001", FormatReplicaID(1))
	require.Equal(t, "00000000000a", FormatReplicaID(10))
	require.Equal(t, "000000000010", FormatReplicaID(16))
}

func TestSequentialReplicaIDs(t *testing.T) {
	config := &mkeconfig.ClusterConfig{
		Spec: &mkeconfig.ClusterSpec{
			Hosts: []*mkeconfig.Host{
				{Role: "msr"},
				{Role: "msr", MSRMetadata: &mkeconfig.MSRMetadata{ReplicaID: "00000000001f"}},
				{Role: "msr"},
			},
			MSR: &mkeconfig.MSRConfig{ReplicaIDs: "sequential"},
		},
	}
	require.NoError(t, AssignSequentialReplicaIDs(config))
	require.Equal(t, "000000000020", config.Spec.Hosts[0].MSRMetadata.ReplicaID)
	require.Equal(t, "00000000001f", config.Spec.Hosts[1].MSRMetadata.ReplicaID)
	require.Equal(t, "000000000021", config.Spec.Hosts[2].MSRMetadata.ReplicaID)
}

// TestParseComposeProjectLabel covers the unchecked indexing of host output.
// The previous code took outputFields[3] and outputFields[len-1] with no
// length check, so a short or empty label panicked.
func TestParseComposeProjectLabel(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantVersion string
		wantReplica string
		wantErr     bool
	}{
		{
			// The shape the existing code assumes: the version is the fourth
			// whitespace-separated field and the replica ID is the last one
			// with a trailing ")" stripped.
			name:        "well formed label",
			output:      "dtr compose project 2.9.10 abc123def)",
			wantVersion: "2.9.10",
			wantReplica: "abc123def",
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			output:  "   \t\n ",
			wantErr: true,
		},
		{
			name:    "one field",
			output:  "dtr",
			wantErr: true,
		},
		{
			name:    "exactly three fields is one short",
			output:  "dtr dtr 2.9.10",
			wantErr: true,
		},
		{
			name:        "four fields is the minimum",
			output:      "a b c 2.9.10",
			wantVersion: "2.9.10",
			wantReplica: "2.9.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				gotVersion, gotReplica string
				err                    error
			)
			require.NotPanics(t, func() {
				gotVersion, gotReplica, err = parseComposeProjectLabel(tt.output)
			})

			if tt.wantErr {
				require.ErrorIs(t, err, errUnexpectedLabel)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantVersion, gotVersion)
			require.Equal(t, tt.wantReplica, gotReplica)
		})
	}
}

// TestBootstrapImageFor covers the negative slice index: LastIndexByte returns
// -1 for an image name with no separator, which panicked.
func TestBootstrapImageFor(t *testing.T) {
	tests := []struct {
		name      string
		imagename string
		version   string
		want      string
	}{
		{
			name:      "registry and namespace",
			imagename: "registry.example.test/mirantis/rethinkdb:2.4",
			version:   "2.9.10",
			want:      "registry.example.test/mirantis/dtr:2.9.10",
		},
		{
			name:      "namespace only",
			imagename: "mirantis/rethinkdb:2.4",
			version:   "2.9.10",
			want:      "mirantis/dtr:2.9.10",
		},
		{
			name:      "bare image name has no repository to derive from",
			imagename: "rethinkdb:2.4",
			version:   "2.9.10",
			want:      "",
		},
		{
			name:      "empty image name",
			imagename: "",
			version:   "2.9.10",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			require.NotPanics(t, func() { got = bootstrapImageFor(tt.imagename, tt.version) })
			require.Equal(t, tt.want, got)
		})
	}
}
