package dtr

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/Mirantis/mcc/pkg/api"
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

func TestBuildUcpFlags(t *testing.T) {
	config := &api.ClusterConfig{
		Spec: &api.ClusterSpec{
			Ucp: api.UcpConfig{
				InstallFlags: []string{
					"--san ucp.acme.com",
					"--admin-username admin",
					"--admin-password password1234",
				},
			},
			Dtr: &api.DtrConfig{},
		},
	}

	t.Run("UCP flags are built when --san is provided", func(t *testing.T) {
		actual := BuildUcpFlags(config)
		expected := []string{
			"--ucp-url ucp.acme.com",
			"--ucp-username admin",
			"--ucp-password 'password1234'",
		}
		sort.Strings(actual)
		sort.Strings(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected is not equal to actual\nexpected: %s\nactual: %s", expected, actual)
		}
	})
}

func TestSequentialReplicaID(t *testing.T) {

	t.Run("A sequential id is generated for up to 9 replicas with a length of 12 characters", func(t *testing.T) {
		for i := 1; i == 9; i++ {
			expected := fmt.Sprintf("0000000000%s", strconv.Itoa(i))
			actual := SequentialReplicaID(i)
			require.Equal(t, expected, actual)
			require.Len(t, actual, 12)
		}
	})
}
