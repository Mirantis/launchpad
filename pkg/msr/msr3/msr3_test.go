package msr3

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

func TestCollectFacts(t *testing.T) {
	t.Run("no MSR CR found", func(t *testing.T) {
		kc := kubeclient.NewTestClient(t)
		rc := kubeclient.NewTestResourceClient(t, kc.Namespace)

		actual, err := CollectFacts(context.Background(), "msr-test", kc, rc, nil)
		assert.NoError(t, err)
		assert.Equal(t, actual, &api.MSRMetadata{Installed: false})
	})
}
