package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestUpgrade(t *testing.T) {
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
