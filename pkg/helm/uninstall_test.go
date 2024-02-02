package helm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestUninstall(t *testing.T) {
	h, _ := NewHelmTestClient(t, nil)
	rd, _ := InstallRethinkDBOperatorChart(t, h)

	err := h.Uninstall(context.Background(), &Options{
		ReleaseDetails: rd,
		Timeout:        ptr.To(DefaultTimeout),
	})
	assert.NoError(t, err)

	rd.ReleaseName = ""
	err = h.Uninstall(context.Background(), &Options{
		ReleaseDetails: rd,
		Timeout:        ptr.To(DefaultTimeout),
	})
	assert.ErrorContains(t, err, "release name is empty")
}
