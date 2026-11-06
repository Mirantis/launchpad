package version

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPickLatestTolerATesUnparseableTags covers the trailing-nil defect: the
// previous implementation sized the slice to len(tags) and advanced its index
// only on a successful parse, so any unparseable name left a nil entry that
// sort.Sort dereferenced. No HTTP failure was needed -- a successful response
// containing one bad tag name was enough.
func TestPickLatestToleratesUnparseableTags(t *testing.T) {
	tests := []struct {
		name    string
		current string
		tags    []string
		want    string
		wantErr bool
	}{
		{
			name:    "unparseable tag among good ones",
			current: "v1.0.0",
			tags:    []string{"v1.2.0", "not-a-version", "v1.3.0"},
			want:    "v1.3.0",
		},
		{
			name:    "every tag unparseable",
			current: "v1.0.0",
			tags:    []string{"nightly", "latest", "stable"},
			wantErr: true,
		},
		{
			name:    "no tags at all",
			current: "v1.0.0",
			tags:    nil,
			wantErr: true,
		},
		{
			name:    "stable current skips prereleases",
			current: "v1.0.0",
			tags:    []string{"v1.2.0", "v1.4.0-rc1", "bad"},
			want:    "v1.2.0",
		},
		{
			name:    "prerelease current accepts a prerelease as latest",
			current: "v1.0.0-beta1",
			tags:    []string{"v1.2.0", "v1.4.0-rc1", "bad"},
			want:    "v1.4.0-rc1",
		},
		{
			name:    "only prereleases and a stable current",
			current: "v1.0.0",
			tags:    []string{"v1.4.0-rc1", "v1.5.0-rc2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := Version
			Version = tt.current
			t.Cleanup(func() { Version = original })

			tags := make([]tag, 0, len(tt.tags))
			for _, name := range tt.tags {
				tags = append(tags, tag{Name: name})
			}

			var (
				got string
				err error
			)
			require.NotPanics(t, func() { got, err = pickLatest(tags, "test") })

			if tt.wantErr {
				require.ErrorIs(t, err, ErrNoRelease)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestGetLatestReportsFailuresAsErrors covers the contract change: every
// failure used to return a bare nil, which callers silently dereferenced. The
// quota-exhaustion case is the one that panicked the functional suite.
func TestGetLatestReportsFailuresAsErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "rate limit exhausted",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.WriteHeader(http.StatusForbidden)
			},
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
		},
		{
			name: "unparseable body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("this is not json"))
			},
		},
		{
			name: "no parseable tags",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`[{"name":"nightly"}]`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			original := githubAPIBase
			githubAPIBase = srv.URL
			t.Cleanup(func() { githubAPIBase = original })

			var (
				release *LaunchpadRelease
				err     error
			)
			require.NotPanics(t, func() { release, err = GetLatest(2 * time.Second) })

			require.ErrorIs(t, err, ErrNoRelease, "a failed lookup must report an error, not a bare nil")
			require.Nil(t, release)
		})
	}
}

// TestGetUpgradeSurvivesALookupFailure pins that an upgrade check never
// propagates the failure: it is advisory and must not block a command.
func TestGetUpgradeSurvivesALookupFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	original := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = original })

	var got *LaunchpadRelease
	require.NotPanics(t, func() { got = GetUpgrade() })
	require.Nil(t, got)
}

// TestLatestTagReadErrorDoesNotPanic covers the wrong-variable bug: the
// body-read failure branch logged err.Error() where err was provably nil,
// so the diagnostic path panicked exactly when it was needed. The server
// declares a longer body than it sends and then closes, forcing a read error.
func TestLatestTagReadErrorDoesNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "4096")
		_, _ = w.Write([]byte(`[{"name":"v1.0.0"}]`))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		panic(http.ErrAbortHandler) // drop the connection mid-body
	}))
	t.Cleanup(srv.Close)

	original := githubAPIBase
	githubAPIBase = srv.URL
	t.Cleanup(func() { githubAPIBase = original })

	var err error
	require.NotPanics(t, func() { _, err = latestTag(2 * time.Second) })
	require.ErrorIs(t, err, ErrNoRelease)
}
