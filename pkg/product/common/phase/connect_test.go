package phase

import (
	"errors"
	"fmt"
	"testing"

	rig "github.com/k0sproject/rig/v2"
	"github.com/k0sproject/rig/v2/protocol"
	"github.com/stretchr/testify/require"
)

// TestConnectRetriesTransientAuthErrors is the regression test for the WinRM
// half of PRODENG-3594.
//
// rig v2.1 wrapped WinRM 401/403 in ErrNonRetryable, so the connect phase
// abandoned a host on its first attempt. A Windows host answers WinRM before
// its provisioning has configured authentication, so that aborted clusters
// which would have come up moments later.
//
// rig v2.2 tags a credential rejection with protocol.ErrAuthFailed and
// deliberately does not wrap it in ErrNonRetryable, so the retry happens
// without launchpad classifying anything itself. These cases use the shapes
// rig produces now; the substring matching they used to need is gone.
func TestConnectRetriesTransientAuthErrors(t *testing.T) {
	ciError := fmt.Errorf("connect: connect: client connect: retry: abort condition reached after 1 attempts: %w",
		fmt.Errorf("%w: create shell: http response error: 401 - invalid content type", protocol.ErrAuthFailed))

	for _, tc := range []struct {
		name  string
		err   error
		retry bool
	}{
		{
			name:  "the exact error smoke-fips failed on",
			err:   ciError,
			retry: true,
		},
		{
			name:  "401 in the library's alternate format",
			err:   fmt.Errorf("%w: create shell: http error 401: unauthorized", protocol.ErrAuthFailed),
			retry: true,
		},
		{
			name:  "403 is also a provisioning race, not a permanent state",
			err:   fmt.Errorf("%w: create shell: http error 403: forbidden", protocol.ErrAuthFailed),
			retry: true,
		},
		{
			name:  "a plain connection failure is still retried",
			err:   errors.New("dial tcp 10.0.0.1:5986: i/o timeout"),
			retry: true,
		},
		{
			name:  "host key mismatch stays non-retryable",
			err:   fmt.Errorf("%w: host key verification failed", rig.ErrNonRetryable),
			retry: false,
		},
		{
			name:  "bad certificates stay non-retryable",
			err:   fmt.Errorf("%w: failed to load certificates", rig.ErrNonRetryable),
			retry: false,
		},
		{
			name:  "misconfigured bastion stays non-retryable",
			err:   fmt.Errorf("%w: bastion connection is not an SSH connection", rig.ErrNonRetryable),
			retry: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.retry, shouldRetryConnect(tc.err))
		})
	}
}
