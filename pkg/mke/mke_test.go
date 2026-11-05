package mke

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
)

func TestVersionGreaterThan(t *testing.T) {
	pairs := [][]string{
		{"1.0.0-beta2", "1.0.0-beta1"},
		{"1.0.0", "1.0.0-rc1"},
		{"1.0.0-rc1", "1.0.0-tp3"},
	}

	for _, pair := range pairs {
		va, _ := version.NewVersion(pair[0])
		vb, _ := version.NewVersion(pair[1])
		require.Truef(t, VersionGreaterThan(va, vb), "%s should be greater than %s", va, vb)
	}
}

// A login failure is routine -- wrong password, unreachable MKE, TLS error --
// and its error reaches the terminal, the debug log and CI logs.
func TestGetClientBundleErrorOmitsPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid credentials"}`))
	}))
	defer server.Close()

	mkeURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	_, err = GetClientBundle(mkeURL, nil, "fake-admin-user", "fake-admin-password")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "fake-admin-password")
	require.Contains(t, err.Error(), "fake-admin-user", "the username identifies which login failed")
}
