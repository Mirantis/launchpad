package msr3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/Mirantis/mcc/pkg/helm"
	"github.com/Mirantis/mcc/pkg/kubeclient"
	"github.com/Mirantis/mcc/pkg/product/mke/api"
)

// prepareTestClient is a helper for returning a KubeClient, ResourceClient and
// HelmClient for testing along with creating an MSR CR that is Ready=true.
func prepareTestClients(t *testing.T, msrVersion string) (*kubeclient.KubeClient, dynamic.ResourceInterface, *helm.Helm) {
	t.Helper()

	kc := kubeclient.NewTestClient(t)
	rc := kubeclient.NewTestResourceClient(t, kc.Namespace)
	h, _ := helm.NewHelmTestClient(t, nil)

	msr := kubeclient.CreateUnstructuredTestMSR(t, msrVersion, true)
	_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
	require.NoError(t, err)

	return kc, rc, h
}

func TestCollectFacts(t *testing.T) {
	t.Run("no MSR CR found", func(t *testing.T) {
		kc := kubeclient.NewTestClient(t)
		rc := kubeclient.NewTestResourceClient(t, kc.Namespace)

		actual, err := CollectFacts(context.Background(), "msr-test", kc, rc, nil)
		assert.NoError(t, err)
		assert.False(t, actual.Installed)
	})

	t.Run("MSR CR found, but not ready", func(t *testing.T) {
		kc := kubeclient.NewTestClient(t)
		rc := kubeclient.NewTestResourceClient(t, kc.Namespace)

		msr := kubeclient.CreateUnstructuredTestMSR(t, "1.0.0", false)

		// CreateUnstructuredTestMSR doesn't populate status.conditions when
		// set to false, we need the status.conditions but we need to set the
		// underlying status to false to simulate the error we want for this
		// test.
		msr.Object["status"] = map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Ready",
					"status": "False",
				},
			},
		}

		_, err := rc.Create(context.Background(), msr, metav1.CreateOptions{})
		require.NoError(t, err)

		_, err = CollectFacts(context.Background(), "msr-test", kc, rc, nil, kubeclient.WithCustomWait(1, time.Millisecond*1))
		assert.ErrorContains(t, err, "MSR CR is not ready")
	})

	t.Run("MSR CR has no version", func(t *testing.T) {
		kc, rc, h := prepareTestClients(t, "")

		_, err := CollectFacts(context.Background(), "msr-test", kc, rc, h)
		assert.ErrorContains(t, err, "unable to determine version")
	})

	t.Run("no helm dependencies", func(t *testing.T) {
		kc, rc, h := prepareTestClients(t, "3.1.1")

		_, err := CollectFacts(context.Background(), "msr-test", kc, rc, h)
		assert.ErrorContains(t, err, "failed to find any installed helm dependencies")
	})

	t.Run("metadata is populated", func(t *testing.T) {
		kc, rc, h := prepareTestClients(t, "3.1.1")

		rd, _ := helm.InstallRethinkDBOperatorChart(t, h)

		actual, err := CollectFacts(context.Background(), "msr-test", kc, rc, h)
		assert.NoError(t, err)

		// We cannot populate ReleaseDetails.RepoURL from a helm.List so we
		// shouldn't expect the InstalledDependencies map to contain this.
		rd.RepoURL = ""

		assert.Equal(t, &api.MSRMetadata{
			Installed:        true,
			InstalledVersion: "3.1.1",
			MSR3: api.MSR3Metadata{
				InstalledDependencies: map[string]helm.ReleaseDetails{
					"rethinkdb-operator": rd,
				},
			}}, actual)
	})
}
