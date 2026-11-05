package helm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/utils/ptr"
)

func TestUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping upgrade test in short mode (requires OCI chart pull)")
	}
	h := NewHelmTestClient(t)

	t.Run("Upgrade success", func(t *testing.T) {
		rd, uninstallFunc := InstallCertManagerChart(t, h)
		t.Cleanup(uninstallFunc)
		rd.Values = map[string]any{ // picking arbitrary values to have set
			"livenessProbe": map[string]any{
				"enabled": true,
			},
		}

		rel, err := h.Upgrade(context.Background(), &Options{
			ReleaseDetails: rd,
			Timeout:        ptr.To(DefaultTimeout),
		})
		if assert.NoError(t, err) {
			assert.Equal(t, rd.ReleaseName, rel.Chart.Metadata.Name)
			assert.ObjectsAreEqualValues(rd.Values, rel.Chart.Values)
		}
	})

	t.Run("Upgrade, reuse values", func(t *testing.T) {
		rd, uninstallFunc := InstallCertManagerChart(t, h)
		t.Cleanup(uninstallFunc)

		rd.Values = map[string]any{
			"image": map[string]any{
				"tag": "1.2.3",
			},
		}

		rel, err := h.Upgrade(context.Background(), &Options{
			ReleaseDetails: rd,
			ReuseValues:    true,
			Timeout:        ptr.To(DefaultTimeout),
		})
		if assert.NoError(t, err) {
			assert.Equal(t, rd.ReleaseName, rel.Chart.Metadata.Name)
			// ReuseValues should not change the values, but reuse the previous
			// ones.
			assert.NotEqual(t, "1.2.3", rel.Chart.Values["image"].(map[string]interface{})["tag"])
			assert.ObjectsAreEqualValues(rd.Values, rel.Chart.Values)
		}
	})

	t.Run("Upgrade failure, empty release details", func(t *testing.T) {
		_, err := h.Upgrade(context.Background(), &Options{
			ReleaseDetails: ReleaseDetails{},
			Timeout:        ptr.To(DefaultTimeout),
		})
		assert.ErrorContains(t, err, "failed to upgrade")
	})

}

// serveTestChartRepo packages a minimal chart at each of the given versions,
// indexes them and serves the result as a Helm chart repository over HTTP.
// It returns the repository URL.
func serveTestChartRepo(t *testing.T, chartName string, versions ...string) string {
	t.Helper()

	repoDir := t.TempDir()
	srv := httptest.NewServer(http.FileServer(http.Dir(repoDir)))
	t.Cleanup(srv.Close)

	index := repo.NewIndexFile()

	for _, ver := range versions {
		meta := &chart.Metadata{APIVersion: chart.APIVersionV2, Name: chartName, Version: ver}

		path, err := chartutil.Save(&chart.Chart{Metadata: meta}, repoDir)
		require.NoError(t, err)

		digest, err := provenance.DigestFile(path)
		require.NoError(t, err)

		require.NoError(t, index.MustAdd(meta, filepath.Base(path), srv.URL, digest))
	}

	index.SortEntries()
	require.NoError(t, index.WriteFile(filepath.Join(repoDir, "index.yaml"), 0o644))

	return srv.URL
}

// newRepoTestClient returns a Helm test client whose repository config and
// cache are scoped to the test, so chart resolution cannot pick up or disturb
// the ambient Helm environment.
func newRepoTestClient(t *testing.T) *Helm {
	t.Helper()

	h := NewHelmTestClient(t)

	helmHome := t.TempDir()
	h.settings.RepositoryConfig = filepath.Join(helmHome, "repositories.yaml")
	h.settings.RepositoryCache = filepath.Join(helmHome, "cache")

	return h
}

// TestUpgradeChartPinning covers that the requested chart version and
// repository are the ones the chart is actually resolved from.  The test
// repository serves three versions and neither the install nor the upgrade
// asks for the newest, so resolving from the ambient environment or falling
// back to "latest" fails the assertions rather than passing by accident.
func TestUpgradeChartPinning(t *testing.T) {
	repoURL := serveTestChartRepo(t, "pinned", "0.1.0", "0.2.0", "0.3.0")
	h := newRepoTestClient(t)

	rd := ReleaseDetails{
		ChartName:   "pinned",
		ReleaseName: "pinned",
		RepoURL:     repoURL,
		Version:     "0.1.0",
	}

	t.Run("install resolves the requested version from the requested repository", func(t *testing.T) {
		rel, err := h.Upgrade(context.Background(), &Options{
			ReleaseDetails: rd,
			Timeout:        ptr.To(DefaultTimeout),
		})
		require.NoError(t, err)

		assert.Equal(t, "0.1.0", rel.Chart.Metadata.Version)
		assert.Equal(t, 1, rel.Version, "expected the install path, not an upgrade")
	})

	t.Run("upgrade resolves the requested version from the requested repository", func(t *testing.T) {
		upgraded := rd
		upgraded.Version = "0.2.0"

		rel, err := h.Upgrade(context.Background(), &Options{
			ReleaseDetails: upgraded,
			Timeout:        ptr.To(DefaultTimeout),
		})
		require.NoError(t, err)

		assert.Equal(t, "0.2.0", rel.Chart.Metadata.Version)
		assert.Equal(t, 2, rel.Version, "expected the upgrade path, not a fresh install")
	})
}

func TestUpgradeProvenanceVerification(t *testing.T) {
	repoURL := serveTestChartRepo(t, "pinned", "0.1.0")

	rd := ReleaseDetails{
		ChartName:   "pinned",
		ReleaseName: "pinned",
		RepoURL:     repoURL,
		Version:     "0.1.0",
	}

	t.Run("Verify without Keyring is rejected", func(t *testing.T) {
		_, err := newRepoTestClient(t).Upgrade(context.Background(), &Options{
			ReleaseDetails: rd,
			Verify:         true,
			Timeout:        ptr.To(DefaultTimeout),
		})
		assert.ErrorIs(t, err, errVerifyWithoutKeyring)
	})

	t.Run("Verify reaches chart resolution", func(t *testing.T) {
		// The test repository publishes no .prov files, so an opted-in
		// verification has to fail at resolution; the same call without
		// Verify succeeds.  That contrast is what shows Verify is carried
		// into LocateChart rather than merely stored.
		keyring := filepath.Join(t.TempDir(), "pubring.gpg")
		require.NoError(t, os.WriteFile(keyring, []byte{}, 0o600))

		verified := newRepoTestClient(t)

		_, err := verified.Upgrade(context.Background(), &Options{
			ReleaseDetails: rd,
			Verify:         true,
			Keyring:        keyring,
			Timeout:        ptr.To(DefaultTimeout),
		})
		assert.ErrorContains(t, err, "failed to locate chart")

		unverified := newRepoTestClient(t)

		_, err = unverified.Upgrade(context.Background(), &Options{
			ReleaseDetails: rd,
			Timeout:        ptr.To(DefaultTimeout),
		})
		assert.NoError(t, err)
	})
}
