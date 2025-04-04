package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestUninstall(t *testing.T) {
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
