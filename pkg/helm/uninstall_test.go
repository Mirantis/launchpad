package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestUninstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping uninstall test in short mode (requires OCI chart pull)")
	}
	h := NewHelmTestClient(t)
	rd, _ := InstallCertManagerChart(t, h)

	err := h.Uninstall(&Options{
		ReleaseDetails: rd,
		Timeout:        ptr.To(DefaultTimeout),
	})
	assert.NoError(t, err)

	rd.ReleaseName = ""
	err = h.Uninstall(&Options{
		ReleaseDetails: rd,
		Timeout:        ptr.To(DefaultTimeout),
	})
	assert.ErrorContains(t, err, "release name is empty")
}
